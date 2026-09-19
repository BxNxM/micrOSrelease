package tui

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m model) handleKey(key string) (tea.Model, tea.Cmd) {
	if (key == "ctrl+c" || key == "q") && m.scanCancel != nil {
		m.scanCancel()
	}
	if key == "ctrl+c" {
		if m.operationCancel != nil {
			m.operationCancel()
		}
		return m, tea.Quit
	}
	if m.showFirmware {
		return m.firmwareKey(key)
	}
	if m.showNodes {
		if m.showNodeDetails {
			return m.deviceDetailsKey(key)
		}
		switch key {
		case "enter", "space":
			if m.nodeIndex == 0 {
				m.showNodes = false
				m.cursor = int(actionDiscovery)
				m.switchBoard(0)
			} else if m.nodeIndex <= len(m.nodes) {
				m.showNodeDetails = true
				m.detailAction = 0
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
	if !m.confirming && (key == "esc" || key == "backspace") {
		refresh := (m.operation == operationInstall || m.operation == operationUpdate) &&
			(m.result != nil || m.operationError != "")
		m.showNodes = true
		m.dismissOperation = true
		if !m.running {
			m.clearOperation()
			if refresh {
				return m.refreshNodes()
			}
		}
		return m, nil
	}
	if m.running || m.loadingInventory || m.probingUSB {
		return m, nil
	}
	if m.confirming {
		switch key {
		case "y", "enter":
			m.confirming = false
			m.running = true
			m.operationContext, m.operationCancel = context.WithCancel(context.Background())
			m.dismissOperation = false
			m.progress = 0
			m.stages = nil
			m.result = nil
			m.operationError = ""
			m.status = strings.ToUpper(string(m.operation[:1])) + string(m.operation[1:]) + " in progress…"
			return m, tea.Batch(m.operationCmd(), tickCmd())
		case "n", "esc", "q":
			m.confirming = false
			m.status = "Operation cancelled"
		}
		return m, nil
	}

	switch key {
	case "q":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(actions)-1 {
			m.cursor++
		}
	case "left", "h":
		m.switchBoard(-1)
	case "right", "l":
		m.switchBoard(1)
	case "[":
		if m.deviceIndex > 0 {
			m.deviceIndex--
			m.selectDeviceBoard()
		}
	case "]":
		if m.deviceIndex+1 < len(m.inventory.Devices) {
			m.deviceIndex++
			m.selectDeviceBoard()
		}
	case "r":
		return m.refreshInventory()
	case "f":
		m.pendingOperation = ""
		m.showFirmware = true
		m.prepareFirmwarePicker()
		return m, nil
	case "enter", "space":
		return m.activate()
	}
	return m, nil
}

func (m model) activate() (tea.Model, tea.Cmd) {
	switch action(m.cursor) {
	case actionRefresh:
		return m.refreshInventory()
	case actionDiscovery:
		m.usbDiscoveryRequested = true
		m.usbDiscoveryFailures = 0
		m.usbScanRequested = true
		m.loadingInventory = true
		m.status = "Discovering USB devices · scanning, then identifying (boards will reset)…"
		return m, m.inventoryCmd()
	case actionInstall:
		return m.confirm(operationInstall)
	case actionUpdate:
		return m.confirm(operationUpdate)
	}
	return m, nil
}

func (m model) refreshInventory() (tea.Model, tea.Cmd) {
	m.usbScanRequested = true
	m.loadingInventory = true
	m.status = "Refreshing USB devices and release images…"
	return m, m.inventoryCmd()
}

func (m model) confirm(requested operation) (tea.Model, tea.Cmd) {
	if !m.firmwareSelected {
		m.pendingOperation = requested
		m.showFirmware = true
		m.prepareFirmwarePicker()
		m.status = "Choose firmware to continue " + string(requested)
		return m, nil
	}
	if len(m.inventory.Devices) == 0 || len(m.inventory.Images) == 0 {
		m.status = "Select a USB device and release image first"
		return m, nil
	}
	m.clearOperation()
	m.operation = requested
	m.confirming = true
	m.result = nil
	return m, nil
}
