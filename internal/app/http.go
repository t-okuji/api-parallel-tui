package app

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
)

func runAllCmd(specs []RequestSpec, concurrency int, client *http.Client) tea.Cmd {
	return func() tea.Msg {
		if concurrency <= 0 {
			concurrency = 1
		}

		events := make(chan runEventMsg)
		ctx, rawCancel := context.WithCancel(context.Background())
		cancel := sync.OnceFunc(rawCancel)

		go func() {
			defer cancel()
			defer close(events)

			sem := make(chan struct{}, concurrency)
			var wg sync.WaitGroup

			for i, spec := range specs {
				wg.Add(1)
				go func(i int, spec RequestSpec) {
					defer wg.Done()

					select {
					case sem <- struct{}{}:
					case <-ctx.Done():
						events <- resultDoneMsg{Index: i, Result: abortedQueuedResult(spec)}
						return
					}
					defer func() {
						<-sem
					}()

					select {
					case <-ctx.Done():
						events <- resultDoneMsg{Index: i, Result: abortedQueuedResult(spec)}
						return
					default:
					}

					events <- resultRunningMsg{Index: i}
					result := executeRequest(ctx, spec, client)
					events <- resultDoneMsg{Index: i, Result: result}
				}(i, spec)
			}

			wg.Wait()
			events <- runCompletedMsg{}
		}()

		return runStartMsg{events: events, cancel: cancel}
	}
}

func waitForRunEventCmd(events chan runEventMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-events
		if !ok {
			return runCompletedMsg{}
		}
		return msg
	}
}

func executeRequest(ctx context.Context, spec RequestSpec, client *http.Client) Result {
	base := Result{
		Name:      spec.Name,
		Method:    spec.Method,
		SourceIdx: spec.SourceIdx,
		RunIndex:  spec.RunIndex,
		RunTotal:  spec.RunTotal,
		Started:   true,
	}

	start := time.Now()

	var body io.Reader
	if len(bytes.TrimSpace(spec.Body)) > 0 {
		body = bytes.NewReader(spec.Body)
	}

	req, err := http.NewRequestWithContext(ctx, spec.Method, spec.URL, body)
	if err != nil {
		base.Duration = time.Since(start)
		base.Err = err
		base.Done = true
		return base
	}

	for k, v := range spec.Headers {
		req.Header.Set(k, v)
	}
	if body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(req)
	if err != nil {
		base.Duration = time.Since(start)
		if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			base.Aborted = true
			base.Err = nil
			base.Done = true
			return base
		}
		base.Err = err
		base.Done = true
		return base
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if readErr != nil {
		base.StatusCode = resp.StatusCode
		base.Duration = time.Since(start)
		base.Err = readErr
		base.Done = true
		return base
	}

	base.StatusCode = resp.StatusCode
	base.Duration = time.Since(start)
	base.Body = string(respBody)
	base.Done = true
	return base
}

func abortedQueuedResult(spec RequestSpec) Result {
	return Result{
		Name:      spec.Name,
		Method:    spec.Method,
		SourceIdx: spec.SourceIdx,
		RunIndex:  spec.RunIndex,
		RunTotal:  spec.RunTotal,
		Aborted:   true,
		Started:   false,
		Done:      true,
	}
}
