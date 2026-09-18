package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/micros/microsctl/internal/network"
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
		var observations []network.Device
		for _, node := range m.nodeObservations {
			if node.CardKey() != msg.uid {
				observations = append(observations, node)
			}
		}
		m.nodeObservations = observations
		for i, node := range m.nodes {
			if node.CardKey() == msg.uid {
				m.nodes = append(m.nodes[:i:i], m.nodes[i+1:]...)
				break
			}
		}
		m.nodeIndex = min(m.nodeIndex, len(m.nodes))
		m.showNodeDetails = false
		m.status = "Device removed from cache · a future scan may rediscover it"
		if msg.hidden {
			m.status = "Special endpoint hidden · the next scan will check it again"
		}
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
		m.nodes = nil
		m.nodeObservations = nil
		for _, node := range msg.nodes {
			if node.SpecialEndpoint() == "" {
				m.applyNetworkDevice(node)
			}
		}
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
		return m.handleInventory(msg)
	case usbProbeMsg:
		return m.handleUSBProbe(msg)
	case networkMsg:
		if msg.device != nil && msg.device.CardKey() != m.removingUID && msg.device.CardKey() != m.removedUID {
			m.applyNetworkDevice(*msg.device)
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
		return m.handleOperation(msg)
	case tickMsg:
		if m.running {
			m.spinnerFrame++
			return m, tickCmd()
		}
	}
	return m, nil
}
