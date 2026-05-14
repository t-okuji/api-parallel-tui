package app

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.ModalMode != modalNone {
		return m.updateModal(msg)
	}

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
			m.openSaveSessionModal()
			m.syncViewportContent()
			return m, nil
		case "ctrl+o":
			sessions, err := listSavedSessions()
			if err != nil {
				m.StatusMessage = "load failed: " + err.Error()
				m.syncViewportContent()
				return m, nil
			}
			if len(sessions) == 0 {
				m.StatusMessage = "no saved sessions"
				m.syncViewportContent()
				return m, nil
			}
			m.openLoadSessionModal(sessions)
			m.syncViewportContent()
			return m, nil
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
			m.moveToAdjacentRequest(1)
			m.updateFocus()
			m.syncViewportContent()
			return m, nil
		case "shift+tab":
			m.moveToAdjacentRequest(-1)
			m.updateFocus()
			m.syncViewportContent()
			return m, nil
		case "up":
			m.moveVertical(-1)
			m.syncViewportContent()
			return m, nil
		case "down":
			m.moveVertical(1)
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

func (m model) updateModal(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.ModalMode {
	case modalSaveSession:
		if key, ok := msg.(tea.KeyPressMsg); ok {
			switch key.String() {
			case "esc":
				m.closeModal()
				m.StatusMessage = "save cancelled"
				m.syncViewportContent()
				return m, nil
			case "enter":
				name := strings.TrimSpace(m.SaveNameInput.Value())
				if name == "" {
					m.StatusMessage = "session name is required"
					m.syncViewportContent()
					return m, nil
				}
				if err := m.saveSession(name); err != nil {
					m.StatusMessage = "save failed: " + err.Error()
					m.syncViewportContent()
					return m, nil
				}
				m.CurrentSession = name
				m.closeModal()
				m.StatusMessage = "saved session: " + name
				m.syncViewportContent()
				return m, nil
			}
		}

		var cmd tea.Cmd
		m.SaveNameInput, cmd = m.SaveNameInput.Update(msg)
		m.syncViewportContent()
		return m, cmd

	case modalLoadSession:
		if key, ok := msg.(tea.KeyPressMsg); ok {
			switch key.String() {
			case "esc":
				m.closeModal()
				m.StatusMessage = "load cancelled"
				m.syncViewportContent()
				return m, nil
			case "up", "k":
				if m.SelectedSession > 0 {
					m.SelectedSession--
				}
				m.syncViewportContent()
				return m, nil
			case "down", "j":
				if m.SelectedSession < len(m.SavedSessions)-1 {
					m.SelectedSession++
				}
				m.syncViewportContent()
				return m, nil
			case "enter":
				if len(m.SavedSessions) == 0 {
					m.closeModal()
					m.syncViewportContent()
					return m, nil
				}
				next, err := loadSessionByID(m.SavedSessions[m.SelectedSession].ID, m.Client)
				if err != nil {
					m.StatusMessage = "load failed: " + err.Error()
					m.closeModal()
					m.syncViewportContent()
					return m, nil
				}
				next.Width = m.Width
				next.Height = m.Height
				next.resizeInputs()
				next.resizeViewport()
				next.syncViewportContent()
				next.StatusMessage = "loaded session: " + next.CurrentSession
				return next, nil
			}
		}
	}

	m.syncViewportContent()
	return m, nil
}

func (m *model) openSaveSessionModal() {
	m.ModalMode = modalSaveSession
	m.SaveNameInput.SetValue(m.defaultSessionName())
	m.SaveNameInput.Focus()
	m.Viewport.GotoTop()
}

func (m *model) openLoadSessionModal(sessions []SavedSession) {
	m.ModalMode = modalLoadSession
	m.SavedSessions = sessions
	m.SelectedSession = 0
	m.SaveNameInput.Blur()
	m.Viewport.GotoTop()
}

func (m *model) closeModal() {
	m.ModalMode = modalNone
	m.SavedSessions = nil
	m.SelectedSession = 0
	m.SaveNameInput.Blur()
	m.updateFocus()
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

func (m *model) moveVertical(delta int) {
	switch m.FocusIndex {
	case fieldConcurrency:
		if delta > 0 {
			m.FocusIndex = fieldRepeat
			m.updateFocus()
		}
	case fieldRepeat:
		if delta < 0 {
			m.FocusIndex = fieldConcurrency
		} else {
			m.FocusIndex = fieldIndexFor(m.ActiveReq, fieldName)
		}
		m.updateFocus()
	default:
		reqIndex, field := decodeFieldIndex(m.FocusIndex)
		if reqIndex < 0 || reqIndex >= len(m.Forms) {
			return
		}

		nextField := field + delta
		switch {
		case nextField < fieldName:
			m.FocusIndex = fieldRepeat
		case nextField > fieldPayload:
			return
		default:
			m.FocusIndex = fieldIndexFor(reqIndex, nextField)
		}
		m.updateFocus()
	}
}

func (m *model) moveToAdjacentRequest(delta int) {
	if len(m.Forms) == 0 {
		return
	}
	if m.FocusIndex < globalFieldCount {
		m.FocusIndex = fieldIndexFor(m.ActiveReq, fieldName)
		return
	}

	reqIndex, field := decodeFieldIndex(m.FocusIndex)
	if reqIndex < 0 || reqIndex >= len(m.Forms) {
		return
	}

	reqIndex += delta
	if reqIndex < 0 {
		reqIndex = len(m.Forms) - 1
	}
	if reqIndex >= len(m.Forms) {
		reqIndex = 0
	}
	m.FocusIndex = fieldIndexFor(reqIndex, field)
}
