package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

type removeDeviceMsg struct {
	uid    string
	err    error
	hidden bool
}

func (m model) deviceDetailsKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "backspace":
		m.showNodeDetails = false
	case "q":
		return m, tea.Quit
	case "up", "k":
		m.detailAction = 0
	case "down", "j":
		m.detailAction = 1
	case "enter", "space", "o":
		if m.nodeIndex <= 0 || m.nodeIndex > len(m.nodes) || m.removingUID != "" {
			return m, nil
		}
		node := m.nodes[m.nodeIndex-1]
		if key == "o" || m.detailAction == 0 {
			if url := node.WebUIURL(); url != "" {
				return m, openBrowserCmd(url)
			}
			return m, nil
		}
		m.removingUID = node.CardKey()
		if node.SpecialEndpoint() != "" {
			m.status = "Hiding special endpoint until the next scan…"
			return m, func() tea.Msg { return removeDeviceMsg{uid: node.CardKey(), hidden: true} }
		}
		m.status = "Removing device from cache…"
		if m.scanCancel != nil {
			m.scanCancel()
		}
		return m, func() tea.Msg {
			remover, ok := m.network.(interface{ RemoveDevice(string) error })
			if !ok {
				return removeDeviceMsg{uid: node.UID, err: fmt.Errorf("device cache removal unavailable")}
			}
			return removeDeviceMsg{uid: node.UID, err: remover.RemoveDevice(node.UID)}
		}
	}
	return m, nil
}
