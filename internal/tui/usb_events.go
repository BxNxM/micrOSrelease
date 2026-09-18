package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/micros/microsctl/internal/usb"
)

func (m model) handleInventory(msg inventoryMsg) (tea.Model, tea.Cmd) {
	m.loadingInventory = false
	if msg.err != nil {
		m.usbDiscoveryRequested = false
		m.usbScanRequested = false
		m.status = "USB refresh failed: " + msg.err.Error()
		return m, nil
	}
	selectedPath := ""
	if m.firmwareSelected && m.imageIndex < len(m.inventory.Images) {
		selectedPath = m.inventory.Images[m.imageIndex].Path
	}
	selectedPort := ""
	if m.deviceIndex >= 0 && m.deviceIndex < len(m.inventory.Devices) {
		selectedPort = m.inventory.Devices[m.deviceIndex].Port
	}
	knownInfo := make(map[string]*usb.DeviceInfo, len(m.inventory.Devices))
	for _, device := range m.inventory.Devices {
		if device.Info != nil {
			knownInfo[device.Port] = device.Info
		}
	}
	for i := range msg.inventory.Devices {
		if !m.usbDiscoveryRequested {
			msg.inventory.Devices[i].Info = knownInfo[msg.inventory.Devices[i].Port]
		} else {
			msg.inventory.Devices[i].Info = nil
		}
	}
	m.inventory = msg.inventory
	if m.usbScanRequested {
		m.usbScanned = true
		m.usbScanRequested = false
	}
	m.deviceIndex, m.imageIndex = 0, 0
	for i, device := range m.inventory.Devices {
		if device.Port == selectedPort {
			m.deviceIndex = i
			break
		}
	}
	m.firmwareSelected = false
	for i, image := range m.inventory.Images {
		if image.Path == selectedPath {
			m.imageIndex = i
			m.firmwareSelected = true
			break
		}
	}
	m.firmwareIndex = m.imageIndex
	boards := usb.Boards(m.inventory.Images)
	found := false
	for _, board := range boards {
		if board == m.boardType {
			found = true
		}
	}
	if !found && len(boards) > 0 {
		m.boardType = boards[0]
	}
	m.prepareFirmwarePicker()
	if !m.showNodes {
		m.status = fmt.Sprintf("Ready: %d USB device(s), %d release image(s)", len(m.inventory.Devices), len(m.inventory.Images))
		m.switchBoard(0)
	}
	if m.usbDiscoveryRequested {
		m.usbDiscoveryRequested = false
		if len(m.inventory.Devices) == 0 {
			m.status = "USB discovery complete · no devices found"
			return m, nil
		}
		m.probingUSB = true
		m.status = "Discovering USB devices · identifying 1/" + fmt.Sprint(len(m.inventory.Devices))
		return m, m.probeUSBCmd(0)
	}
	return m, nil
}

func (m model) handleUSBProbe(msg usbProbeMsg) (tea.Model, tea.Cmd) {
	if !m.probingUSB {
		return m, nil
	}
	if msg.err != nil || msg.device.Info == nil || msg.device.Info.Chip == "" {
		m.usbDiscoveryFailures++
	} else if msg.index >= 0 && msg.index < len(m.inventory.Devices) && m.inventory.Devices[msg.index].Port == msg.device.Port {
		m.inventory.Devices[msg.index] = msg.device
	}
	if next := msg.index + 1; next < len(m.inventory.Devices) {
		m.status = fmt.Sprintf("Discovering USB devices · identifying %d/%d", next+1, len(m.inventory.Devices))
		return m, m.probeUSBCmd(next)
	}
	m.probingUSB = false
	if m.deviceIndex >= len(m.inventory.Devices) || m.inventory.Devices[m.deviceIndex].Info == nil {
		for i, device := range m.inventory.Devices {
			if device.Info != nil {
				m.deviceIndex = i
				break
			}
		}
	}
	matched := m.selectDeviceBoard()
	m.status = fmt.Sprintf("USB discovery complete · %d identified", len(m.inventory.Devices)-m.usbDiscoveryFailures)
	if m.usbDiscoveryFailures > 0 {
		m.status += fmt.Sprintf(" · %d could not be identified", m.usbDiscoveryFailures)
	}
	if matched {
		m.status += " · framework: " + m.boardType
	} else {
		m.status += " · select a matching framework manually"
	}
	return m, nil
}
