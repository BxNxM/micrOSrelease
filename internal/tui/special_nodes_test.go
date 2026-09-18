package tui

import (
	"testing"

	"github.com/micros/microsctl/internal/network"
	"github.com/micros/microsctl/internal/tui/widgets"
)

func TestSpecialCardsPreserveDistinctDevicesAndSelection(t *testing.T) {
	lan := network.Device{Address: "192.168.1.10:9008", UID: "lan", Online: true}
	ap := network.Device{Address: network.APAddress, UID: "ap", Online: true}
	local := network.Device{Name: "simulator", Address: network.LocalhostAddress, UID: "local", Online: true}
	m := model{nodes: []network.Device{lan}, nodeIndex: 1, showNodeDetails: true}
	m.applyNetworkDevice(ap)
	m.applyNetworkDevice(local)
	if len(m.nodes) != 3 || m.nodeIndex != 3 || !m.showNodeDetails {
		t.Fatal("inserting distinct special devices lost the selected LAN device")
	}
	ap.Online = false
	m.applyNetworkDevice(ap)
	if len(m.nodes) != 2 || m.nodeIndex != 2 || m.nodes[1].Address != lan.Address {
		t.Fatal("offline AP must disappear without removing or deselecting a different LAN device")
	}
	m.nodeIndex = 1
	local.Online = false
	m.applyNetworkDevice(local)
	if len(m.nodes) != 1 || m.showNodeDetails || m.nodeIndex != 1 {
		t.Fatal("offline selected localhost must close details and leave other devices intact")
	}
}

func TestSimulatorUIDMergesAliasesRegardlessOfScanOrder(t *testing.T) {
	const uid = "micr303862363166336236643138OS"
	local := network.Device{Name: "simulator", Address: network.LocalhostAddress, UID: uid, Online: true}
	lan := network.Device{Name: "simulator", Address: "192.168.1.10:9008", UID: uid, Online: true}
	ap := network.Device{Name: "node01", Address: network.APAddress, UID: uid, Online: true}
	for _, order := range [][]network.Device{{lan, ap, local}, {local, ap, lan}, {ap, local, lan}} {
		m := model{}
		for i, device := range order {
			m.applyNetworkDevice(device)
			if i == 0 {
				m.nodeIndex, m.showNodeDetails = 1, true
			}
			if len(m.nodes) != 1 || m.nodes[0].UID != uid || m.nodeIndex != 1 || !m.showNodeDetails {
				t.Fatalf("UID was duplicated or selection lost: %+v", m.nodes)
			}
		}
		if m.nodes[0].Address != local.Address || widgets.NodeTitle(m.nodes[0]) != "localhost" {
			t.Fatalf("localhost did not win: %+v", m.nodes)
		}
		// An offline result from a duplicate endpoint must not remove localhost.
		lan.Online = false
		m.applyNetworkDevice(lan)
		if len(m.nodes) != 1 || m.nodes[0].Address != local.Address {
			t.Fatal("offline LAN replaced localhost")
		}
		ap.Online = false
		m.applyNetworkDevice(ap)
		// Failed probes may have only an address, with no hello/UID available.
		m.applyNetworkDevice(network.Device{Address: network.LocalhostAddress})
		if len(m.nodes) != 0 || m.showNodeDetails {
			t.Fatalf("offline simulator aliases survived: %+v", m.nodes)
		}
		lan.Online, ap.Online = true, true
	}
}

func TestUnavailableSimulatorNeverAppears(t *testing.T) {
	m := model{}
	for _, device := range []network.Device{
		{Name: "simulator", Address: network.LocalhostAddress, UID: "sim-uid"},
		{Name: "simulator", Address: network.LocalhostAddress, UID: "sim-uid", Online: true, Cached: true},
		{Name: "simulator", Address: "192.168.1.10:9008", UID: "sim-uid", Online: true, Cached: true},
	} {
		m.applyNetworkDevice(device)
		if len(m.nodes) != 0 {
			t.Fatalf("unavailable simulator appeared: %+v", m.nodes)
		}
	}
}

func TestHideSpecialCardAlsoHidesLANAlias(t *testing.T) {
	local := network.Device{Address: network.LocalhostAddress, UID: "shared", Online: true}
	lan := network.Device{Address: "192.168.1.10:9008", UID: "shared", Online: true}
	m := model{}
	m.applyNetworkDevice(lan)
	m.applyNetworkDevice(local)
	m.nodeIndex, m.showNodeDetails, m.detailAction = 1, true, 1
	updated, cmd := m.deviceDetailsKey("enter")
	if cmd == nil {
		t.Fatal("hide did not return a command")
	}
	m = updated.(model)
	updated, _ = m.Update(cmd())
	m = updated.(model)
	if len(m.nodes) != 0 || m.removedUID != local.UID {
		t.Fatalf("hide retained alias: %+v", m.nodes)
	}
	updated, _ = m.Update(networkMsg{device: &lan, done: true})
	m = updated.(model)
	if len(m.nodes) != 0 {
		t.Fatal("same scan restored hidden UID through LAN alias")
	}
}

func TestUnavailableAPFallsBackToReachableLAN(t *testing.T) {
	lan := network.Device{Address: "192.168.1.10:9008", UID: "shared", Online: true}
	ap := network.Device{Address: network.APAddress, UID: "shared", Online: true}
	m := model{}
	m.applyNetworkDevice(lan)
	m.applyNetworkDevice(ap)
	ap.Online = false
	m.applyNetworkDevice(ap)
	if len(m.nodes) != 1 || m.nodes[0].Address != lan.Address {
		t.Fatalf("reachable LAN fallback lost: %+v", m.nodes)
	}
}
