package tui

import (
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
)

func (m model) shellKey(key string) (tea.Model, tea.Cmd) {
	if m.shell.busy && key != "esc" && key != "ctrl+c" && key != "left" && key != "right" {
		return m, nil
	}
	switch key {
	case "esc", "ctrl+c":
		m.shell.visible = false
		if m.shell.cancel != nil {
			m.shell.cancel()
		}
		session := m.shell.session
		m.shell.session = nil
		return m, func() tea.Msg {
			if session != nil {
				session.Close()
			}
			return nil
		}
	case "left", "right":
		m.scrollShell(key)
	case "up", "down":
		m.moveShellHistory(key)
	case "enter":
		return m.submitShell()
	case "backspace":
		input := []rune(m.shell.input)
		if len(input) > 0 {
			m.shell.input = string(input[:len(input)-1])
		}
	case "ctrl+u":
		m.shell.input = ""
	case "space":
		m.appendShellInput(" ")
	default:
		if len([]rune(key)) == 1 {
			m.appendShellInput(key)
		}
	}
	return m, nil
}

func (m *model) appendShellInput(text string) {
	if m.shell.busy {
		return
	}
	for _, r := range text {
		if !unicode.IsControl(r) && len(m.shell.input) < 4096 {
			m.shell.input += string(r)
		}
	}
}

// Offset zero is the unfinished input after the newest history entry.
func (m *model) moveShellHistory(key string) {
	if m.shell.session == nil || strings.Contains(m.shell.prompt, "[password]") || len(m.shell.history) == 0 {
		return
	}
	if key == "up" {
		if m.shell.historyOffset == 0 {
			m.shell.draft = m.shell.input
		}
		m.shell.historyOffset = min(len(m.shell.history), m.shell.historyOffset+1)
	} else {
		if m.shell.historyOffset == 0 {
			return
		}
		m.shell.historyOffset--
	}
	if m.shell.historyOffset == 0 {
		m.shell.input = m.shell.draft
	} else {
		m.shell.input = m.shell.history[len(m.shell.history)-m.shell.historyOffset]
	}
}

func (m *model) scrollShell(key string) {
	state := m.renderState()
	limit := max(0, len(state.ShellLines())-state.ShellViewportHeight())
	offset := min(m.shell.scroll, limit)
	step := max(1, state.ShellViewportHeight()/2)
	if key == "left" {
		offset += step
	} else {
		offset -= step
	}
	m.shell.scroll = max(0, min(offset, limit))
}

func (m *model) preserveShellScroll(previousLines int) {
	if m.shell.scroll == 0 {
		return
	}
	state := m.renderState()
	lines := len(state.ShellLines())
	m.shell.scroll = max(0, min(m.shell.scroll+lines-previousLines, max(0, lines-state.ShellViewportHeight())))
}
