package tui

import tea "charm.land/bubbletea/v2"

func (m model) nodesKey(key string) (tea.Model, tea.Cmd) {
	if m.showNodeDetails {
		return m.deviceDetailsKey(key)
	}
	switch key {
	case "ctrl+f":
		m.nodeFilterEditing = true
	case "esc":
		m.nodeFilter = ""
		m.rebuildNodes()
	case "u":
		return m.requestAppUpdate()
	case "enter", "space":
		if m.nodeIndex == 0 {
			m.showNodes = false
			m.cursor = int(actionDiscovery)
			m.switchBoard(0)
		} else if m.nodeIndex <= len(m.nodes) {
			m.showNodeDetails = true
			m.detailAction = 0
			if m.nodes[m.nodeIndex-1].WebUIURL() == "" {
				m.detailAction = 1
			}
		}
	case "q":
		if m.operationCancel != nil {
			m.operationCancel()
		}
		return m, tea.Quit
	case "down", "j", "up", "k", "left", "h", "right", "l":
		m.moveNode(key)
	case "r":
		return m.refreshNodes()
	}
	return m, nil
}

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
