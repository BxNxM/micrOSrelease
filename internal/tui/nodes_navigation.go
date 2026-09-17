package tui

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
