package tui

import (
	"strings"
	"testing"

	"github.com/micros/microsctl/internal/network"
)

func filterKey(t *testing.T, m model, key string) model {
	t.Helper()
	next, cmd := m.handleKey(key)
	if cmd != nil {
		t.Fatalf("filter key %q unexpectedly ran a command", key)
	}
	return next.(model)
}

func TestNodeFilterNavigationAndRefresh(t *testing.T) {
	m := model{showNodes: true, width: 80, height: 30}
	for _, name := range []string{"Kitchen", "Bedroom", "Kitchenette"} {
		m.applyNetworkDevice(network.Device{Name: name, UID: name, Address: name, Online: true})
	}
	m.nodeIndex = 2 // Kitchen
	m = filterKey(t, m, "ctrl+f")
	for _, key := range []string{"K", "I", "T"} {
		m = filterKey(t, m, key)
	}
	if len(m.nodes) != 2 || m.nodes[m.nodeIndex-1].Name != "Kitchen" {
		t.Fatalf("filtered selection: %+v", m.nodes)
	}
	if !strings.Contains(m.View().Content, "Filter: KIT") {
		t.Fatal("filter missing from view")
	}
	m.applyNetworkDevice(network.Device{Name: "Kit board", UID: "new", Address: "new", Online: true})
	m.applyNetworkDevice(network.Device{Name: "Office", UID: "office", Address: "office", Online: true})
	if len(m.nodes) != 3 || len(m.nodeObservations) != 5 {
		t.Fatal("refresh lost filter or hidden nodes")
	}
	m = filterKey(t, m, "enter")
	if !m.showNodeDetails || m.nodes[m.nodeIndex-1].Name != "Kitchen" {
		t.Fatal("opened wrong filtered device")
	}
	m.showNodeDetails = false
	m = filterKey(t, m, "esc")
	if len(m.nodes) != 5 || m.nodeFilter != "" {
		t.Fatal("clear did not restore nodes")
	}
}

func TestNodeFilterEditingConsumesShortcuts(t *testing.T) {
	m := model{showNodes: true, nodes: []network.Device{{Name: "qrx", UID: "one"}}}
	m = filterKey(t, m, "ctrl+f")
	for _, key := range []string{"q", "r", "x"} {
		m = filterKey(t, m, key)
	}
	if m.nodeFilter != "qrx" {
		t.Fatal(m.nodeFilter)
	}
	m = filterKey(t, m, "backspace")
	if m.nodeFilter != "qr" {
		t.Fatal(m.nodeFilter)
	}
	m = filterKey(t, m, "ctrl+u")
	m.appendNodeFilter("Étage")
	m = filterKey(t, m, "backspace")
	if m.nodeFilter != "Étag" || len(m.nodes) != 0 || m.nodeIndex != 0 {
		t.Fatal("unicode or empty matches mishandled")
	}
	m = filterKey(t, m, "esc")
	if m.nodeFilterEditing || len(m.nodes) != 1 {
		t.Fatal("escape did not clear")
	}
}

func TestNodeFilterKeepsGridControlsActive(t *testing.T) {
	m := model{showNodes: true, width: 80, height: 30}
	for _, name := range []string{"node-a", "node-b", "node-c"} {
		m.applyNetworkDevice(network.Device{Name: name, UID: name, Online: true})
	}
	m = filterKey(t, m, "ctrl+f")
	m.appendNodeFilter("node")
	for _, step := range []struct {
		key   string
		index int
	}{
		{"right", 1}, {"down", 3}, {"left", 2}, {"up", 0}, {"right", 1},
	} {
		m = filterKey(t, m, step.key)
		if m.nodeIndex != step.index || !m.nodeFilterEditing || m.nodeFilter != "node" {
			t.Fatalf("%s: index=%d editing=%v filter=%q", step.key, m.nodeIndex, m.nodeFilterEditing, m.nodeFilter)
		}
	}
	m = filterKey(t, m, "enter")
	if !m.showNodeDetails || !m.nodeFilterEditing || m.nodes[m.nodeIndex-1].Name != "node-a" {
		t.Fatal("Enter failed to open selection while filtering")
	}
	m = filterKey(t, m, "esc")
	if m.showNodeDetails || !m.nodeFilterEditing {
		t.Fatal("return from details lost filter editing")
	}
	m = filterKey(t, m, "backspace")
	if m.nodeFilter != "nod" {
		t.Fatal("filter not editable after returning")
	}
	m.nodeIndex = 0
	m = filterKey(t, m, "enter")
	if m.showNodes || m.nodeFilter != "nod" {
		t.Fatal("USB Tools failed to open while filtering")
	}
}
