package app

import (
	"net/http"
	"strconv"
	"strings"
)

func (m model) collectSpecs() []RequestSpec {
	repeat := m.repeatCount()
	specs := make([]RequestSpec, 0, len(m.Forms)*repeat)

	for i, form := range m.Forms {
		base := formToSpec(form)
		base.SourceIdx = i
		base.RunTotal = repeat

		for run := 1; run <= repeat; run++ {
			spec := base
			spec.RunIndex = run
			specs = append(specs, spec)
		}
	}

	return specs
}

func (m model) concurrency() int {
	n, err := strconv.Atoi(strings.TrimSpace(m.ConcurrencyInput.Value()))
	if err != nil || n <= 0 {
		return 1
	}
	return n
}

func (m model) repeatCount() int {
	n, err := strconv.Atoi(strings.TrimSpace(m.RepeatInput.Value()))
	if err != nil || n <= 0 {
		return 1
	}
	return n
}

func parseHeaders(raw string) map[string]string {
	headers := map[string]string{}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key != "" {
			headers[key] = value
		}
	}
	return headers
}

func formToSpec(form RequestForm) RequestSpec {
	method := strings.ToUpper(strings.TrimSpace(form.MethodInput.Value()))
	if method == "" {
		method = http.MethodGet
	}

	return RequestSpec{
		Name:    strings.TrimSpace(form.NameInput.Value()),
		Method:  method,
		URL:     strings.TrimSpace(form.URLInput.Value()),
		Headers: parseHeaders(form.HeadersArea.Value()),
		Body:    []byte(form.PayloadArea.Value()),
	}
}
