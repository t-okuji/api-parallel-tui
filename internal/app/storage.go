package app

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
)

func (m model) saveToFile(path string) error {
	state := persistState{
		Concurrency: m.concurrency(),
		Repeat:      m.repeatCount(),
		Requests:    make([]persistRequest, 0, len(m.Forms)),
	}

	for _, form := range m.Forms {
		state.Requests = append(state.Requests, persistRequest{
			Name:    form.NameInput.Value(),
			Method:  form.MethodInput.Value(),
			URL:     form.URLInput.Value(),
			Headers: form.HeadersArea.Value(),
			Payload: form.PayloadArea.Value(),
		})
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

func loadFromFile(path string, client *http.Client) (model, error) {
	// #nosec G304 -- loading a user-selected local file is intentional here.
	data, err := os.ReadFile(path)
	if err != nil {
		return model{}, err
	}

	var state persistState
	if err := json.Unmarshal(data, &state); err != nil {
		return model{}, err
	}

	m := initialModel()
	m.Client = client
	m.Forms = nil
	m.Results = nil

	if state.Concurrency <= 0 {
		state.Concurrency = 1
	}
	if state.Repeat <= 0 {
		state.Repeat = 1
	}

	m.ConcurrencyInput.SetValue(strconv.Itoa(state.Concurrency))
	m.RepeatInput.SetValue(strconv.Itoa(state.Repeat))

	for _, req := range state.Requests {
		form := newRequestForm()
		form.NameInput.SetValue(req.Name)
		form.MethodInput.SetValue(req.Method)
		form.URLInput.SetValue(req.URL)
		form.HeadersArea.SetValue(req.Headers)
		form.PayloadArea.SetValue(req.Payload)
		m.Forms = append(m.Forms, form)
	}

	if len(m.Forms) == 0 {
		m.Forms = []RequestForm{newRequestForm()}
	}

	m.FocusIndex = fieldConcurrency
	m.ActiveReq = 0
	m.SelectedResult = 0
	return m, nil
}
