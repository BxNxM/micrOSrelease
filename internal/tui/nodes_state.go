package tui

import (
	"sort"
	"strings"

	"github.com/micros/microsctl/internal/network"
)

// nodeRank keeps special endpoints first, followed by release, other, and offline nodes.
// USB Tools occupies index zero separately from the device list.
func nodeRank(node network.Device) int {
	if !node.Online {
		return 4
	}
	switch node.SpecialEndpoint() {
	case "Localhost":
		return 0
	case "AP mode":
		return 1
	}
	if node.Mode == "rel" {
		return 2
	}
	return 3
}

func (m *model) applyNetworkDevice(device network.Device) {
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
	m.rebuildNodes()
}

func (m *model) rebuildNodes() {
	selected := ""
	if m.nodeIndex > 0 && m.nodeIndex <= len(m.nodes) {
		selected = m.nodes[m.nodeIndex-1].CardKey()
	}
	if m.nodeObservations == nil {
		m.nodeObservations = append([]network.Device{}, m.nodes...)
	}
	m.nodes = nil
	for _, node := range network.UniqueDevices(m.nodeObservations) {
		if node.CardKey() == m.removedUID || node.CardKey() == m.removingUID ||
			(node.SpecialEndpoint() != "" && (!node.Online || node.Cached)) {
			continue
		}
		if !strings.Contains(strings.ToLower(node.Name), strings.ToLower(m.nodeFilter)) {
			continue
		}
		m.nodes = append(m.nodes, node)
	}
	sort.SliceStable(m.nodes, func(i, j int) bool {
		a, b := m.nodes[i], m.nodes[j]
		if aRank, bRank := nodeRank(a), nodeRank(b); aRank != bRank {
			return aRank < bRank
		}
		if aName, bName := strings.ToLower(a.Name), strings.ToLower(b.Name); aName != bName {
			return aName < bName
		}
		return a.CardKey() < b.CardKey()
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
