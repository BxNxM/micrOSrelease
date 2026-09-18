package tui

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/micros/microsctl/internal/usb"
)

type discoveryManager struct {
	inventory    usb.Inventory
	inventoryErr error
	chips        map[string]string
	calls        []string
}

func (m *discoveryManager) Inventory(context.Context) (usb.Inventory, error) {
	m.calls = append(m.calls, "scan")
	return m.inventory, m.inventoryErr
}
func (m *discoveryManager) Probe(_ context.Context, d usb.Device) (usb.Device, error) {
	m.calls = append(m.calls, "identify "+d.Port)
	chip := m.chips[d.Port]
	if chip == "" {
		return d, errors.New("not an ESP device")
	}
	d.Info = &usb.DeviceInfo{Chip: chip}
	return d, nil
}
func (*discoveryManager) Install(context.Context, usb.Target, ...func([]usb.Stage)) (usb.Result, error) {
	panic("discovery must not install")
}
func (*discoveryManager) Update(context.Context, usb.Target, ...func([]usb.Stage)) (usb.Result, error) {
	panic("discovery must not update")
}

func TestUSBDiscoveryActionOrderAndDefault(t *testing.T) {
	var titles []string
	for _, a := range actions {
		titles = append(titles, a.title)
	}
	want := []string{"Discovery", "Install micrOS", "Update micrOS", "USB Scan"}
	if !reflect.DeepEqual(titles, want) {
		t.Fatalf("actions=%v", titles)
	}
	m := New(nil, nil)
	m.cursor = int(actionInstall)
	next, _ := m.handleKey("enter")
	if next.(model).cursor != int(actionDiscovery) {
		t.Fatal("opening Actions must select Discovery")
	}
}

func TestUSBDiscoveryScansIdentifiesAndSelectsFramework(t *testing.T) {
	manager := &discoveryManager{
		inventory: usb.Inventory{Devices: []usb.Device{{Port: "A"}, {Port: "B"}}, Images: []usb.Image{
			{Board: "esp32", Path: "esp32.bin"}, {Board: "esp32c6", Path: "c6.bin"},
		}}, chips: map[string]string{"A": "ESP32", "B": "ESP32-C6"},
	}
	m := model{usb: manager, boardType: "esp32", firmwareSelected: true, deviceIndex: 1, inventory: usb.Inventory{
		Devices: []usb.Device{{Port: "A"}, {Port: "B", Info: &usb.DeviceInfo{Chip: "ESP32-S3"}}},
		Images:  manager.inventory.Images,
	}}
	next, cmd := m.activate()
	m = next.(model)
	if cmd == nil || !m.loadingInventory || !m.usbDiscoveryRequested {
		t.Fatal("Discovery did not start a scan")
	}
	for cmd != nil {
		next, cmd = m.Update(cmd())
		m = next.(model)
	}
	if !reflect.DeepEqual(manager.calls, []string{"scan", "identify A", "identify B"}) {
		t.Fatalf("calls=%v", manager.calls)
	}
	if m.deviceIndex != 1 || m.boardType != "esp32c6" || !m.firmwareSelected || m.imageIndex != 1 || m.probingUSB || m.loadingInventory {
		t.Fatalf("discovery did not select detected framework: device=%d board=%q image=%d selected=%v status=%q", m.deviceIndex, m.boardType, m.imageIndex, m.firmwareSelected, m.status)
	}
	next, _ = m.handleKey("[")
	m = next.(model)
	if m.boardType != "esp32" || m.imageIndex != 0 {
		t.Fatal("switching identified device did not update framework")
	}
}

func TestUSBScanDoesNotIdentifyOrChangeFramework(t *testing.T) {
	manager := &discoveryManager{inventory: usb.Inventory{Devices: []usb.Device{{Port: "A"}}, Images: []usb.Image{{Board: "esp32", Path: "esp.bin"}}}}
	m := model{usb: manager, cursor: int(actionRefresh), boardType: "esp32"}
	next, cmd := m.activate()
	m = next.(model)
	next, cmd = m.Update(cmd())
	m = next.(model)
	if cmd != nil || !reflect.DeepEqual(manager.calls, []string{"scan"}) || m.boardType != "esp32" {
		t.Fatal("plain USB Scan must remain passive")
	}
}

func TestUSBDiscoveryHandlesEmptyScanAndFailures(t *testing.T) {
	for _, tc := range []struct {
		name       string
		devices    []usb.Device
		scanErr    error
		wantCalls  []string
		wantStatus string
	}{
		{"empty", nil, nil, []string{"scan"}, "no devices found"},
		{"scan error", nil, errors.New("scan failed"), []string{"scan"}, "scan failed"},
		{"probe error", []usb.Device{{Port: "A"}, {Port: "B"}}, nil, []string{"scan", "identify A", "identify B"}, "1 could not be identified"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manager := &discoveryManager{inventory: usb.Inventory{Devices: tc.devices, Images: []usb.Image{{Board: "esp32c6", Path: "c6.bin"}}}, inventoryErr: tc.scanErr, chips: map[string]string{"B": "ESP32-C6"}}
			m := model{usb: manager}
			next, cmd := m.activate()
			m = next.(model)
			for cmd != nil {
				next, cmd = m.Update(cmd())
				m = next.(model)
			}
			if !reflect.DeepEqual(manager.calls, tc.wantCalls) || !strings.Contains(m.status, tc.wantStatus) || m.probingUSB || m.loadingInventory || m.usbDiscoveryRequested {
				t.Fatalf("calls=%v status=%q", manager.calls, m.status)
			}
			if len(tc.devices) > 1 && m.inventory.Devices[1].Info == nil {
				t.Fatal("probe failure prevented identifying later devices")
			}
			if len(tc.devices) > 1 && (m.deviceIndex != 1 || m.boardType != "esp32c6" || !m.firmwareSelected) {
				t.Fatal("discovery should select an identified device when the previous selection failed")
			}
		})
	}
}

func TestDetectedBoardKeepsFirmwareChoiceUnambiguous(t *testing.T) {
	for _, tc := range []struct {
		chip    string
		images  []usb.Image
		board   string
		matched bool
		latest  int
	}{
		{"ESP32-S3", []usb.Image{{Board: "esp32"}, {Board: "esp32s3", Version: "3.9.0"}, {Board: "esp32s3", Version: "3.10.0"}}, "esp32s3", true, 2},
		{"ESP32-C5", []usb.Image{{Board: "esp32"}}, "esp32", false, 0},
	} {
		m := model{boardType: "esp32", firmwareSelected: true, inventory: usb.Inventory{Devices: []usb.Device{{Info: &usb.DeviceInfo{Chip: tc.chip}}}, Images: tc.images}}
		got := m.selectDeviceBoard()
		if got != tc.matched || m.boardType != tc.board || m.firmwareSelected != tc.matched || (tc.matched && m.imageIndex != tc.latest) {
			t.Fatalf("chip=%s matched=%v board=%s selected=%v image=%d", tc.chip, got, m.boardType, m.firmwareSelected, m.imageIndex)
		}
	}
}
