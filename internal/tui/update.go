package tui

import (
	"fmt"
	"github.com/micros/micros-release/internal/usb"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case removeDeviceMsg:
		m.removingUID = ""
		if msg.err != nil {
			m.status = "Could not remove device: " + msg.err.Error()
			return m, nil
		}
		m.removedUID = msg.uid
		for i, node := range m.nodes {
			if node.UID == msg.uid {
				m.nodes = append(m.nodes[:i:i], m.nodes[i+1:]...)
				break
			}
		}
		m.nodeIndex = min(m.nodeIndex, len(m.nodes))
		m.showNodeDetails = false
		m.status = "Device removed from cache · a future scan may rediscover it"
		return m, nil
	case browserMsg:
		m.status = "Opened Web UI in default browser"
		if msg.err != nil {
			m.status = "Could not open browser: " + msg.err.Error()
		}
	case autoRefreshMsg:
		// Keep exactly one timer chain, and never overlap discovery runs.
		if m.discovering || m.removingUID != "" {
			return m, autoRefreshCmd()
		}
		m.discovering = true
		m.status = "Refreshing nodes in background…"
		cmd := m.networkCmd()
		return m, tea.Batch(cmd, autoRefreshCmd())
	case cachedNodesMsg:
		if msg.err != nil {
			m.status = "Cache unavailable: " + msg.err.Error()
			return m, func() tea.Msg { return startScanMsg{} }
		}
		if len(msg.nodes) == 0 {
			return m.Update(startScanMsg{})
		}
		m.nodes = msg.nodes
		m.status = fmt.Sprintf("Loaded %d saved nodes · refreshing…", len(m.nodes))
		return m, func() tea.Msg { return startScanMsg{} }
	case startScanMsg:
		if !m.discovering {
			m.discovering = true
			m.status = "Scanning TCP 9008…"
			cmd := m.networkCmd()
			return m, cmd
		}
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		return m.handleKey(msg.String())
	case inventoryMsg:
		m.loadingInventory = false
		if msg.err != nil {
			m.status = "USB refresh failed: " + msg.err.Error()
			return m, nil
		}
		selectedPath := ""
		if m.firmwareSelected && m.imageIndex < len(m.inventory.Images) {
			selectedPath = m.inventory.Images[m.imageIndex].Path
		}
		m.inventory = msg.inventory
		if m.usbScanRequested {
			m.usbScanned = true
			m.usbScanRequested = false
		}
		m.deviceIndex, m.imageIndex = 0, 0
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
	case networkMsg:
		if msg.device != nil && msg.device.UID != m.removingUID && msg.device.UID != m.removedUID {
			index := len(m.nodes)
			for i, node := range m.nodes {
				if node.UID == msg.device.UID {
					index = i
					break
				}
			}
			if index == len(m.nodes) {
				m.nodes = append(m.nodes, *msg.device)
			} else {
				m.nodes[index] = *msg.device
			}
			m.status = fmt.Sprintf("Scanning TCP 9008 · %d known nodes", len(m.nodes))
		}
		if !msg.done {
			return m, nextNode(msg.queue)
		}
		m.discovering = false
		m.scanCancel = nil
		m.status = fmt.Sprintf("Scan complete · %d known nodes", len(m.nodes))
		if msg.err != nil {
			m.status = "Scan stopped: " + msg.err.Error()
		}
	case operationMsg:
		if !msg.done {
			m.stages = msg.stages
			completed := 0
			for _, stage := range m.stages {
				if stage.State == usb.StageDone {
					completed++
				}
			}
			m.progress = completed * 100 / max(1, len(m.stages))
			return m, nextOperation(msg.queue)
		}
		m.running = false
		if m.dismissOperation {
			m.clearOperation()
			return m, nil
		}
		if msg.err != nil {
			m.status = "Operation failed: " + msg.err.Error()
			return m, nil
		}
		m.result = &msg.result
		m.progress = 100
		m.status = msg.result.Summary
	case tickMsg:
		if m.running {
			m.spinnerFrame++
			return m, tickCmd()
		}
	}
	return m, nil
}

func (m model) handleKey(key string) (tea.Model, tea.Cmd) {
	if (key == "ctrl+c" || key == "q") && m.scanCancel != nil {
		m.scanCancel()
	}
	if key == "ctrl+c" {
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
				m.switchBoard(0)
			} else if m.nodeIndex <= len(m.nodes) {
				m.showNodeDetails = true
				m.detailAction = 0
			}
		case "q":
			return m, tea.Quit
		case "down", "j", "up", "k", "left", "h", "right", "l":
			m.moveNode(key)
		case "r":
			if !m.discovering && m.removingUID == "" {
				m.discovering = true
				m.status = "Scanning TCP 9008…"
				cmd := m.networkCmd()
				return m, cmd
			}
		}
		return m, nil
	}
	if !m.confirming && (key == "esc" || key == "backspace") {
		m.showNodes = true
		m.dismissOperation = true
		if !m.running {
			m.clearOperation()
		}
		return m, nil
	}
	if m.running || m.loadingInventory {
		return m, nil
	}
	if m.confirming {
		switch key {
		case "y", "enter":
			m.confirming = false
			m.running = true
			m.dismissOperation = false
			m.progress = 0
			m.stages = nil
			m.result = nil
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
		}
	case "]":
		if m.deviceIndex+1 < len(m.inventory.Devices) {
			m.deviceIndex++
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
	m.operation = requested
	m.confirming = true
	m.result = nil
	return m, nil
}
