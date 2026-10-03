package tui

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/micros/microsctl/internal/network"
)

func TestNodesOrderIndependentOfDiscoveryOrder(t *testing.T) {
	nodes := []network.Device{
		{UID: "local", Address: network.LocalhostAddress, Name: "simulator", Mode: "dev", Online: true},
		{UID: "ap", Address: network.APAddress, Name: "node01", Mode: "dev", Online: true},
		{UID: "rel-a", Name: "alpha", Mode: "rel", Online: true},
		{UID: "rel-b", Name: "Beta", Mode: "rel", Online: true},
		{UID: "rel-c", Name: "beta", Mode: "rel", Online: true},
		{UID: "dev-a", Name: "aardvark", Mode: "dev", Online: true},
		{UID: "unknown", Name: "Bravo", Mode: "n/a", Online: true},
		{UID: "dev-z", Name: "zulu", Mode: "dev", Online: true},
		{UID: "offline-a", Name: "alpha", Mode: "dev", Cached: true},
		{UID: "offline-b", Name: "Beta", Mode: "rel", Cached: true},
	}
	want := []string{"local", "ap", "rel-a", "rel-b", "rel-c", "dev-a", "unknown", "dev-z", "offline-a", "offline-b"}
	rng := rand.New(rand.NewSource(1))
	for trial := 0; trial < 30; trial++ {
		m := model{}
		for _, i := range rng.Perm(len(nodes)) {
			m.applyNetworkDevice(nodes[i])
		}
		var got []string
		for _, node := range m.nodes {
			got = append(got, node.UID)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("trial %d: order = %v, want %v", trial, got, want)
		}
		if m.nodeIndex != 0 {
			t.Fatal("discovery moved selection away from USB Tools")
		}
	}
}

func TestNodesReorderPreservesSelectedDevice(t *testing.T) {
	m := model{showNodes: true}
	selected := network.Device{UID: "selected", Name: "zulu", Mode: "dev", Online: true}
	for _, node := range []network.Device{
		{UID: "release", Name: "bravo", Mode: "rel", Online: true},
		{UID: "development", Name: "charlie", Mode: "dev", Online: true},
		selected,
	} {
		m.applyNetworkDevice(node)
	}
	m.nodeIndex, m.showNodeDetails = 3, true
	for _, change := range []struct {
		name, mode string
		online     bool
		wantIndex  int
	}{
		{"alpha", "dev", true, 2},
		{"alpha", "rel", true, 1},
		{"alpha", "rel", false, 3},
		{"alpha", "rel", true, 1},
	} {
		selected.Name, selected.Mode, selected.Online = change.name, change.mode, change.online
		next, _ := m.Update(networkMsg{device: &selected, done: true})
		m = next.(model)
		if m.nodeIndex != change.wantIndex || !m.showNodeDetails || m.nodes[m.nodeIndex-1].UID != selected.UID {
			t.Fatalf("change %+v lost selection: index=%d details=%v", change, m.nodeIndex, m.showNodeDetails)
		}
	}
	// Opening the reordered card must still target the selected device.
	m.showNodeDetails = false
	next, _ := m.handleKey("enter")
	m = next.(model)
	if !m.showNodeDetails || m.nodes[m.nodeIndex-1].UID != selected.UID {
		t.Fatal("opening the reordered card selected the wrong device")
	}
}
