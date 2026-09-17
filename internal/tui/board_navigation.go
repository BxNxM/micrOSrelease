package tui

import "github.com/micros/micros-release/internal/usb"

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
	if m.firmwareSelected && m.inventory.Images[m.imageIndex].Board != m.boardType {
		m.firmwareSelected = false
	}
	m.prepareFirmwarePicker()
	if indices := usb.BoardImages(m.inventory.Images, m.boardType); len(indices) == 1 {
		m.imageIndex = indices[0]
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
	if m.firmwareSelected {
		m.firmwareIndex = m.imageIndex
	}
}
