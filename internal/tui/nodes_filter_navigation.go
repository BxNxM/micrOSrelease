package tui

import (
	tea "charm.land/bubbletea/v2"
	"unicode"
)

func (m model) nodeFilterKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.nodeFilter = ""
		m.nodeFilterEditing = false
		m.rebuildNodes()
	case "left", "right", "up", "down":
		m.moveNode(key)
	case "enter":
		return m.nodesKey(key)
	case "ctrl+f":
		m.nodeFilterEditing = false
	case "backspace":
		input := []rune(m.nodeFilter)
		if len(input) > 0 {
			m.nodeFilter = string(input[:len(input)-1])
			m.rebuildNodes()
		}
	case "ctrl+u":
		m.nodeFilter = ""
		m.rebuildNodes()
	case "space":
		m.appendNodeFilter(" ")
	default:
		if len([]rune(key)) == 1 {
			m.appendNodeFilter(key)
		}
	}
	return m, nil
}

func (m *model) appendNodeFilter(text string) {
	for _, r := range text {
		if !unicode.IsControl(r) && len([]rune(m.nodeFilter)) < 128 {
			m.nodeFilter += string(r)
		}
	}
	m.rebuildNodes()
}
