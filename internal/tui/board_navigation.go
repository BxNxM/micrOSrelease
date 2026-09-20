package tui

import "github.com/micros/microsctl/internal/usb"

// Discovery follows the selected device, while manual board navigation remains
// available for board variants and chips without a bundled framework.
func (m *model) selectDeviceBoard() bool {
	if m.deviceIndex < 0 || m.deviceIndex >= len(m.inventory.Devices) {
		return false
	}
	info := m.inventory.Devices[m.deviceIndex].Info
	if info == nil || info.Chip == "" {
		return false
	}
	if board, matched := usb.BoardForChip(m.inventory.Images, info.Chip); matched {
		m.boardType = board
		m.switchBoard(0)
		if latest, found := usb.LatestBoardImage(m.inventory.Images, board); found {
			m.imageIndex = latest
			m.firmwareIndex = latest
			m.firmwareSelected = true
			m.status = "Selected latest firmware: " + m.inventory.Images[latest].Name
		}
		return true
	}
	// Do not leave an incompatible image selected after identifying a new chip.
	m.firmwareSelected = false
	m.pendingOperation = ""
	return false
}

func (m *model) switchBoard(direction int) {
	boards := usb.Boards(m.inventory.Images)
	if len(boards) == 0 {
		return
	}
	index := 0
	for i, board := range boards {
		if board == m.boardType {
			index = i
			break
		}
	}
	m.boardType = boards[(index+direction+len(boards))%len(boards)]
	m.prepareFirmwarePicker()
	if m.showFirmware {
		return
	}
	if m.firmwareSelected && m.inventory.Images[m.imageIndex].Board == m.boardType {
		return
	}
	m.firmwareSelected = false
	if indices := usb.BoardImages(m.inventory.Images, m.boardType); len(indices) == 1 || (direction != 0 && len(indices) > 0) {
		m.imageIndex = indices[0]
		m.firmwareIndex = m.imageIndex
		m.firmwareSelected = true
		m.status = "Selected firmware: " + m.inventory.Images[m.imageIndex].Name
	}
}

func (m *model) prepareFirmwarePicker() {
	indices := usb.BoardImages(m.inventory.Images, m.boardType)
	if len(indices) == 0 {
		m.firmwareIndex = 0
		return
	}
	m.firmwareIndex = indices[0]
	if m.firmwareSelected && m.inventory.Images[m.imageIndex].Board == m.boardType {
		m.firmwareIndex = m.imageIndex
	}
}
