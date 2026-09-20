package tui

import (
	"sort"

	"github.com/micros/microsctl/internal/network"
)

func specialRank(node network.Device) int {
	switch node.SpecialEndpoint() {
	case "Localhost":
		return 0
	case "AP mode":
		return 1
	default:
		return 2
	}
}

func (m *model) applyNetworkDevice(device network.Device) {
	selected := ""
	if m.nodeIndex > 0 && m.nodeIndex <= len(m.nodes) {
		selected = m.nodes[m.nodeIndex-1].CardKey()
	}
	if m.nodeObservations == nil {
		m.nodeObservations = append([]network.Device{}, m.nodes...)
	}
	index := len(m.nodeObservations)
	for i, node := range m.nodeObservations {
		if (device.Address != "" && node.Address == device.Address) ||
			(device.Address == "" && node.CardKey() == device.CardKey()) {
			index = i
			if device.UID == "" && !device.Online {
				device.UID = node.UID
			}
			break
		}
	}
	if index == len(m.nodeObservations) {
		m.nodeObservations = append(m.nodeObservations, device)
	} else {
		m.nodeObservations[index] = device
	}
	m.nodes = nil
	for _, node := range network.UniqueDevices(m.nodeObservations) {
		if node.CardKey() == m.removedUID || node.CardKey() == m.removingUID ||
			(node.SpecialEndpoint() != "" && (!node.Online || node.Cached)) {
			continue
		}
		m.nodes = append(m.nodes, node)
	}
	sort.SliceStable(m.nodes, func(i, j int) bool {
		if m.nodes[i].Online != m.nodes[j].Online {
			return m.nodes[i].Online
		}
		return specialRank(m.nodes[i]) < specialRank(m.nodes[j])
	})
	if selected != "" {
		for i, node := range m.nodes {
			if node.CardKey() == selected {
				m.nodeIndex = i + 1
				return
			}
		}
		m.showNodeDetails = false
	}
	m.nodeIndex = min(m.nodeIndex, len(m.nodes))
}
