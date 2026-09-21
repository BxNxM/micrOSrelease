package tui

import (
	"testing"

	"github.com/micros/microsctl/internal/network"
)

func TestDeviceDetailsDefaultSelection(t *testing.T) {
	for _, tc := range []struct {
		name string
		node network.Device
		want int
	}{
		{"enabled", network.Device{Name: "node", Features: map[string]string{"webui": "ON"}}, 0},
		{"disabled", network.Device{Name: "node", Features: map[string]string{"webui": "OFF"}}, 1},
		{"unknown", network.Device{Name: "node"}, 1},
		{"no usable URL", network.Device{Features: map[string]string{"webui": "ON"}}, 1},
	} {
		for _, key := range []string{"enter", "space"} {
			t.Run(tc.name+"/"+key, func(t *testing.T) {
				m := model{showNodes: true, nodeIndex: 1, nodes: []network.Device{tc.node}, detailAction: 2}
				next, _ := m.handleKey(key)
				m = next.(model)
				if !m.showNodeDetails || m.detailAction != tc.want {
					t.Fatalf("details=%v action=%d want=%d", m.showNodeDetails, m.detailAction, tc.want)
				}
			})
		}
	}
}

func TestOfflineShellCannotOpen(t *testing.T) {
	for _, key := range []string{"enter", "space"} {
		m := model{showNodes: true, showNodeDetails: true, nodeIndex: 1, detailAction: 1, nodes: []network.Device{{Address: "192.168.1.2:9008"}}}
		next, cmd := m.handleKey(key)
		if next.(model).shell.visible || cmd != nil {
			t.Fatalf("%s opened offline shell", key)
		}
	}
}
