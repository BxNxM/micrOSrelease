package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/micros/microsctl/internal/usb"
)

func (m model) firmwareKey(key string) (tea.Model, tea.Cmd) {
	indices := usb.BoardImages(m.inventory.Images, m.boardType)
	position := 0
	for i, index := range indices {
		if index == m.firmwareIndex {
			position = i
		}
	}
	switch key {
	case "left", "h":
		m.switchBoard(-1)
	case "right", "l":
		m.switchBoard(1)
	case "esc", "backspace":
		m.showFirmware = false
		if m.firmwareSelected {
			m.boardType = m.inventory.Images[m.imageIndex].Board
			m.firmwareIndex = m.imageIndex
		}
		if m.pendingOperation != "" {
			m.status = "Operation cancelled"
			m.pendingOperation = ""
		}
	case "q":
		return m, tea.Quit
	case "up", "k":
		if len(indices) > 0 {
			m.firmwareIndex = indices[max(0, position-1)]
		}
	case "down", "j":
		if len(indices) > 0 {
			m.firmwareIndex = indices[min(len(indices)-1, position+1)]
		}
	case "enter", "space":
		if len(indices) > 0 {
			m.imageIndex = m.firmwareIndex
			m.firmwareSelected = true
			m.boardType = m.inventory.Images[m.imageIndex].Board
			m.showFirmware = false
			m.status = "Selected firmware: " + m.inventory.Images[m.imageIndex].Name
			if requested := m.pendingOperation; requested != "" {
				m.pendingOperation = ""
				return m.confirm(requested)
			}
		}
	}
	return m, nil
}
