package tui

import tea "charm.land/bubbletea/v2"

// refreshNodes shares the manual refresh guards with post-USB navigation.
func (m model) refreshNodes() (tea.Model, tea.Cmd) {
	if m.discovering || m.removingUID != "" {
		return m, nil
	}
	m.discovering = true
	m.status = "Scanning TCP 9008…"
	cmd := m.networkCmd()
	return m, cmd
}

// moveNode follows grid geometry without wrapping horizontally into another row.
func (m *model) moveNode(key string) {
	if len(m.nodes) == 0 {
		return
	}
	_, columns, _ := m.nodeGrid()
	switch key {
	case "left", "h":
		if m.nodeIndex%columns > 0 {
			m.nodeIndex--
		}
	case "right", "l":
		if m.nodeIndex%columns < columns-1 && m.nodeIndex < len(m.nodes) {
			m.nodeIndex++
		}
	case "up", "k":
		if m.nodeIndex >= columns {
			m.nodeIndex -= columns
		}
	case "down", "j":
		if (m.nodeIndex/columns+1)*columns <= len(m.nodes) {
			m.nodeIndex = min(m.nodeIndex+columns, len(m.nodes))
		}
	}
}
