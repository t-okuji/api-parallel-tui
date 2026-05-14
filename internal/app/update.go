package app

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.resizeInputs()
		m.resizeViewport()
		m.syncViewportContent()
		return m, nil

	case allResultsMsg:
		m.Running = false
		m.Results = msg.Results
		m.clampSelectedResult()
		m.syncActiveRequestToSelectedResult()
		m.StatusMessage = fmt.Sprintf("completed %d execution(s)", len(msg.Results))
		m.syncViewportContent()
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.syncViewportContent()
			return m, tea.Quit
		case "ctrl+r":
			specs := m.collectSpecs()
			if len(specs) == 0 {
				m.StatusMessage = "no requests to run"
				m.syncViewportContent()
				return m, nil
			}

			m.Running = true
			m.Results = make([]Result, len(specs))
			m.SelectedResult = 0
			for i, spec := range specs {
				m.Results[i] = Result{
					Running:   true,
					Name:      spec.Name,
					Method:    spec.Method,
					SourceIdx: spec.SourceIdx,
					RunIndex:  spec.RunIndex,
					RunTotal:  spec.RunTotal,
				}
			}
			m.StatusMessage = fmt.Sprintf(
				"running %d execution(s) from %d request(s) with parallelism=%d",
				len(specs),
				len(m.Forms),
				m.concurrency(),
			)
			m.syncViewportContent()
			return m, runAllCmd(specs, m.concurrency(), m.Client)
		case "ctrl+n":
			m.Forms = append(m.Forms, newRequestForm())
			m.ActiveReq = len(m.Forms) - 1
			m.FocusIndex = fieldIndexFor(m.ActiveReq, fieldName)
			m.resizeInputs()
			m.updateFocus()
			m.StatusMessage = fmt.Sprintf("request %d added", len(m.Forms))
			m.syncViewportContent()
			return m, nil
		case "ctrl+d":
			if len(m.Forms) == 1 {
				m.StatusMessage = "at least one request is required"
				m.syncViewportContent()
				return m, nil
			}
			m.Forms = append(m.Forms[:m.ActiveReq], m.Forms[m.ActiveReq+1:]...)
			if m.ActiveReq >= len(m.Forms) {
				m.ActiveReq = len(m.Forms) - 1
			}
			m.FocusIndex = fieldIndexFor(m.ActiveReq, fieldName)
			m.resizeInputs()
			m.updateFocus()
			m.StatusMessage = fmt.Sprintf("request %d deleted", m.ActiveReq+1)
			m.syncViewportContent()
			return m, nil
		case "ctrl+s":
			if err := m.saveToFile(requestsFile); err != nil {
				m.StatusMessage = "save failed: " + err.Error()
				m.syncViewportContent()
				return m, nil
			}
			m.StatusMessage = "saved to " + requestsFile
			m.syncViewportContent()
			return m, nil
		case "ctrl+o":
			next, err := loadFromFile(requestsFile, m.Client)
			if err != nil {
				m.StatusMessage = "load failed: " + err.Error()
				m.syncViewportContent()
				return m, nil
			}
			next.Width = m.Width
			next.Height = m.Height
			next.resizeInputs()
			next.resizeViewport()
			next.updateFocus()
			next.StatusMessage = "loaded from " + requestsFile
			next.syncViewportContent()
			return next, nil
		case "ctrl+j":
			if len(m.Results) > 0 && m.SelectedResult < len(m.Results)-1 {
				m.SelectedResult++
				m.syncActiveRequestToSelectedResult()
			}
			m.syncViewportContent()
			return m, nil
		case "ctrl+k":
			if len(m.Results) > 0 && m.SelectedResult > 0 {
				m.SelectedResult--
				m.syncActiveRequestToSelectedResult()
			}
			m.syncViewportContent()
			return m, nil
		case "tab":
			m.FocusIndex = (m.FocusIndex + 1) % m.totalFields()
			m.updateFocus()
			m.syncViewportContent()
			return m, nil
		case "shift+tab":
			m.FocusIndex--
			if m.FocusIndex < 0 {
				m.FocusIndex = m.totalFields() - 1
			}
			m.updateFocus()
			m.syncViewportContent()
			return m, nil
		case "up":
			if m.FocusIndex >= globalFieldCount && m.ActiveReq > 0 {
				m.ActiveReq--
				m.FocusIndex = fieldIndexFor(m.ActiveReq, currentField(m.FocusIndex))
				m.updateFocus()
			}
			m.syncViewportContent()
			return m, nil
		case "down":
			if m.FocusIndex >= globalFieldCount && m.ActiveReq < len(m.Forms)-1 {
				m.ActiveReq++
				m.FocusIndex = fieldIndexFor(m.ActiveReq, currentField(m.FocusIndex))
				m.updateFocus()
			}
			m.syncViewportContent()
			return m, nil
		case "pgdown", "ctrl+f":
			m.Viewport.PageDown()
			m.syncViewportContent()
			return m, nil
		case "pgup", "ctrl+b":
			m.Viewport.PageUp()
			m.syncViewportContent()
			return m, nil
		}
	}

	if _, ok := msg.(tea.MouseMsg); ok {
		var cmd tea.Cmd
		m.Viewport, cmd = m.Viewport.Update(msg)
		m.syncViewportContent()
		return m, cmd
	}

	var cmd tea.Cmd
	m, cmd = m.updateFocusedInput(msg)
	m.syncViewportContent()
	return m, cmd
}

func (m *model) updateFocus() {
	m.ConcurrencyInput.Blur()
	m.RepeatInput.Blur()
	for i := range m.Forms {
		m.Forms[i].NameInput.Blur()
		m.Forms[i].MethodInput.Blur()
		m.Forms[i].URLInput.Blur()
		m.Forms[i].HeadersArea.Blur()
		m.Forms[i].PayloadArea.Blur()
	}

	switch m.FocusIndex {
	case fieldConcurrency:
		m.ConcurrencyInput.Focus()
		return
	case fieldRepeat:
		m.RepeatInput.Focus()
		return
	}

	reqIndex, field := decodeFieldIndex(m.FocusIndex)
	if reqIndex < 0 || reqIndex >= len(m.Forms) {
		return
	}

	m.ActiveReq = reqIndex
	switch field {
	case fieldName:
		m.Forms[reqIndex].NameInput.Focus()
	case fieldMethod:
		m.Forms[reqIndex].MethodInput.Focus()
	case fieldURL:
		m.Forms[reqIndex].URLInput.Focus()
	case fieldHeaders:
		m.Forms[reqIndex].HeadersArea.Focus()
	case fieldPayload:
		m.Forms[reqIndex].PayloadArea.Focus()
	}
}

func (m model) updateFocusedInput(msg tea.Msg) (model, tea.Cmd) {
	switch m.FocusIndex {
	case fieldConcurrency:
		var cmd tea.Cmd
		m.ConcurrencyInput, cmd = m.ConcurrencyInput.Update(msg)
		return m, cmd
	case fieldRepeat:
		var cmd tea.Cmd
		m.RepeatInput, cmd = m.RepeatInput.Update(msg)
		return m, cmd
	}

	reqIndex, field := decodeFieldIndex(m.FocusIndex)
	if reqIndex < 0 || reqIndex >= len(m.Forms) {
		return m, nil
	}

	var cmd tea.Cmd
	form := m.Forms[reqIndex]
	switch field {
	case fieldName:
		form.NameInput, cmd = form.NameInput.Update(msg)
	case fieldMethod:
		form.MethodInput, cmd = form.MethodInput.Update(msg)
	case fieldURL:
		form.URLInput, cmd = form.URLInput.Update(msg)
	case fieldHeaders:
		form.HeadersArea, cmd = form.HeadersArea.Update(msg)
	case fieldPayload:
		form.PayloadArea, cmd = form.PayloadArea.Update(msg)
	}
	m.Forms[reqIndex] = form
	return m, cmd
}
