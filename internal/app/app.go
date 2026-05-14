package app

import (
	"net/http"
	"time"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

func NewProgram() *tea.Program {
	return tea.NewProgram(initialModel())
}

func newRequestForm() RequestForm {
	name := textinput.New()
	method := textinput.New()
	url := textinput.New()
	headers := textarea.New()
	payload := textarea.New()

	name.SetValue("")
	method.SetValue(http.MethodGet)
	url.SetValue("")
	headers.SetValue("Content-Type: application/json")
	payload.SetValue("")

	headers.SetHeight(5)
	payload.SetHeight(8)

	return RequestForm{
		NameInput:   name,
		MethodInput: method,
		URLInput:    url,
		HeadersArea: headers,
		PayloadArea: payload,
	}
}

func initialModel() model {
	concurrency := textinput.New()
	concurrency.SetValue("3")

	repeat := textinput.New()
	repeat.SetValue("1")

	m := model{
		ConcurrencyInput: concurrency,
		RepeatInput:      repeat,
		Forms:            []RequestForm{newRequestForm()},
		Results:          []Result{},
		Viewport:         viewport.New(),
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
		StatusMessage: "ctrl+n: add  ctrl+d: delete  ctrl+r: run  ctrl+j/k: select result  pgup/pgdn: scroll  ctrl+s: save  ctrl+o: load  q: quit",
	}
	m.Viewport.SoftWrap = true
	m.Viewport.MouseWheelEnabled = true
	m.Viewport.MouseWheelDelta = 3
	m.updateFocus()
	m.syncViewportContent()
	return m
}

func (m model) Init() tea.Cmd {
	return nil
}
