package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

func (m model) View() tea.View {
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	status := statusStyle.Render(m.StatusMessage)

	var screen strings.Builder
	screen.WriteString(m.Viewport.View())
	screen.WriteString("\n")
	screen.WriteString(status)

	v := tea.NewView(screen.String())
	v.AltScreen = true
	if m.MouseModeEnabled {
		v.MouseMode = tea.MouseModeCellMotion
	}
	return v
}

func (m *model) resizeInputs() {
	width := m.Width - 22
	if width < 20 {
		width = 20
	}

	m.ConcurrencyInput.SetWidth(8)
	m.RepeatInput.SetWidth(8)
	for i := range m.Forms {
		m.Forms[i].NameInput.SetWidth(width)
		m.Forms[i].MethodInput.SetWidth(12)
		m.Forms[i].URLInput.SetWidth(width)
		m.Forms[i].HeadersArea.SetWidth(width)
		m.Forms[i].PayloadArea.SetWidth(width)
	}
}

func (m *model) resizeViewport() {
	width := m.Width
	if width < 20 {
		width = 20
	}

	height := m.Height - 1
	if height < 1 {
		height = 1
	}

	m.Viewport.SetWidth(width)
	m.Viewport.SetHeight(height)
}

func (m *model) syncViewportContent() {
	m.Viewport.SetContent(m.renderContent())
}

func (m model) renderContent() string {
	var b strings.Builder

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Render("API Concurrent Runner")
	b.WriteString(title)
	b.WriteString("\n\n")

	b.WriteString(m.renderGlobalSection())
	b.WriteString("\n")

	if modal := m.renderModal(); modal != "" {
		b.WriteString(modal)
		b.WriteString("\n")
	}

	for i, form := range m.Forms {
		b.WriteString(m.renderRequestSection(i, form))
		b.WriteString("\n")
	}

	b.WriteString(m.renderResultsSection())
	return b.String()
}

func (m model) renderGlobalSection() string {
	label := lipgloss.NewStyle().Bold(true).Render("[Global]")
	return strings.Join([]string{
		label,
		"Parallelism: " + m.ConcurrencyInput.View(),
		"Repeat:      " + m.RepeatInput.View(),
		"Session:     " + m.currentSessionLabel(),
	}, "\n")
}

func (m model) renderModal() string {
	switch m.ModalMode {
	case modalSaveSession:
		return lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("86")).
			Padding(0, 1).
			Render(strings.Join([]string{
				"[Save Session]",
				"Name: " + m.SaveNameInput.View(),
				"enter: save  esc: cancel",
			}, "\n"))
	case modalLoadSession:
		lines := []string{"[Load Session]"}
		for i, session := range m.SavedSessions {
			cursor := " "
			if i == m.SelectedSession {
				cursor = ">"
			}
			lines = append(lines, fmt.Sprintf("%s %s (%d requests, %s)", cursor, session.Name, session.RequestCount, session.UpdatedAt))
		}
		lines = append(lines, "enter: load  up/down: select  esc: cancel")
		return lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("86")).
			Padding(0, 1).
			Render(strings.Join(lines, "\n"))
	default:
		return ""
	}
}

func (m model) renderRequestSection(index int, form RequestForm) string {
	headerStyle := lipgloss.NewStyle().Bold(true)
	activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

	header := fmt.Sprintf("[Request %d]", index+1)
	if index == m.ActiveReq {
		header = activeStyle.Render(header + " (active)")
	} else {
		header = headerStyle.Render(header)
	}

	lines := []string{
		header,
		"Name:    " + form.NameInput.View(),
		"Method:  " + form.MethodInput.View(),
		"URL:     " + form.URLInput.View(),
		"Headers:",
		form.HeadersArea.View(),
		"",
		"Payload:",
		form.PayloadArea.View(),
		"",
		"Payload Preview:",
		renderPayloadPreview(form.PayloadArea.Value()),
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Render(strings.Join(lines, "\n"))
}

func (m model) renderResultsSection() string {
	lines := []string{lipgloss.NewStyle().Bold(true).Render("[Results]")}

	if len(m.Results) == 0 {
		lines = append(lines, "No executions yet")
	} else {
		for reqIndex := range m.Forms {
			if section, ok := m.renderRequestResultSection(reqIndex); ok {
				lines = append(lines, section)
			}
		}
	}

	if active, ok := m.activeResult(); ok {
		if active.Done && strings.TrimSpace(active.Body) != "" {
			lines = append(lines, "", "Body Preview:", m.renderBodyPreview(active))
		} else if active.Err != nil {
			lines = append(lines, "", "Error:", active.Err.Error())
		}
	}

	return strings.Join(lines, "\n")
}

func (m model) renderRequestResultSection(reqIndex int) (string, bool) {
	resultIndexes := make([]int, 0)
	for i, result := range m.Results {
		if result.SourceIdx == reqIndex {
			resultIndexes = append(resultIndexes, i)
		}
	}

	if len(resultIndexes) == 0 {
		return "", false
	}

	lines := []string{
		lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("86")).
			Render(fmt.Sprintf("Request %d", reqIndex+1)),
	}

	name := strings.TrimSpace(m.Forms[reqIndex].NameInput.Value())
	if name != "" {
		lines = append(lines, "Name: "+name)
	}

	for _, i := range resultIndexes {
		cursor := " "
		if i == m.SelectedResult {
			cursor = ">"
		}
		lines = append(lines, fmt.Sprintf(
			"%s %s %-28s %s",
			cursor,
			m.resultIcon(m.Results[i]),
			m.resultDisplayName(m.Results[i]),
			m.resultSummary(m.Results[i].Method, m.Results[i]),
		))
	}

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Render(strings.Join(lines, "\n")), true
}

func (m model) resultIcon(r Result) string {
	switch {
	case r.Queued:
		return "…"
	case r.Running:
		return "⏳"
	case r.Aborted && r.Started:
		return "⏹"
	case r.Aborted:
		return "⊘"
	case r.Err != nil:
		return "❌"
	case r.Done:
		return "✅"
	default:
		return "•"
	}
}

func (m model) resultSummary(method string, r Result) string {
	switch {
	case r.Queued:
		return "queued"
	case r.Running:
		return "running..."
	case r.Aborted && r.Started:
		return "aborted while running"
	case r.Aborted:
		return "aborted before start"
	case r.Err != nil:
		return r.Err.Error()
	case r.Done:
		return fmt.Sprintf("%s %d %s", method, r.StatusCode, r.Duration.Round(time.Millisecond))
	default:
		return "not started"
	}
}

func (m model) runningStatus() string {
	var queued, running, done, abortedRunning, abortedQueued int
	for _, result := range m.Results {
		switch {
		case result.Queued:
			queued++
		case result.Running:
			running++
		case result.Aborted && result.Started:
			abortedRunning++
		case result.Aborted:
			abortedQueued++
		case result.Done:
			done++
		}
	}
	return fmt.Sprintf("running=%d queued=%d completed=%d aborted_running=%d aborted_queued=%d total=%d", running, queued, done, abortedRunning, abortedQueued, len(m.Results))
}

func (m model) cancelledStatus() string {
	var completed, abortedRunning, abortedQueued int
	for _, result := range m.Results {
		switch {
		case result.Aborted && result.Started:
			abortedRunning++
		case result.Aborted:
			abortedQueued++
		case result.Done:
			completed++
		}
	}
	return fmt.Sprintf("aborted run: completed=%d aborted_running=%d aborted_queued=%d total=%d", completed, abortedRunning, abortedQueued, len(m.Results))
}

func (m model) resultDisplayName(result Result) string {
	name := strings.TrimSpace(result.Name)
	if name == "" {
		name = fmt.Sprintf("Request %d", result.SourceIdx+1)
	}
	if result.RunTotal <= 1 {
		return name
	}
	return fmt.Sprintf("%s #%d/%d", name, result.RunIndex, result.RunTotal)
}

func (m model) activeResult() (Result, bool) {
	if len(m.Results) == 0 {
		return Result{}, false
	}
	if m.SelectedResult < 0 || m.SelectedResult >= len(m.Results) {
		return Result{}, false
	}
	return m.Results[m.SelectedResult], true
}

func (m model) renderBodyPreview(result Result) string {
	title := fmt.Sprintf("%s response", m.resultDisplayName(result))
	body := formatBody(result.Body)

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Render(title + "\n" + body)
}

func formatBody(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	var pretty bytes.Buffer
	if json.Valid([]byte(raw)) && json.Indent(&pretty, []byte(raw), "", "  ") == nil {
		return highlightJSON(pretty.String())
	}
	return raw
}

func renderPayloadPreview(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			Render("(empty)")
	}

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		Render(formatBody(trimmed))
}

func highlightJSON(source string) string {
	lexer := lexers.Get("json")
	if lexer == nil {
		return source
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, source)
	if err != nil {
		return source
	}

	var out bytes.Buffer
	if err := formatter.Format(&out, style, iterator); err != nil {
		return source
	}

	return strings.TrimRight(out.String(), "\n")
}
