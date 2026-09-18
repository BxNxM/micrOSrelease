package tui

import (
	"testing"

	"github.com/micros/microsctl/internal/usb"
)

func TestInventoryRefreshPreservesIdentifiedDeviceInfo(t *testing.T) {
	info := &usb.DeviceInfo{Chip: "ESP32-C6", FlashSize: "4 MB"}
	m := model{
		inventory: usb.Inventory{Devices: []usb.Device{{Port: "/dev/ttyACM0", Info: info}}},
		showNodes: true,
	}

	updated, _ := m.Update(inventoryMsg{inventory: usb.Inventory{
		Devices: []usb.Device{{Port: "/dev/ttyACM0"}, {Port: "/dev/ttyUSB0"}},
	}})
	got := updated.(model)
	if got.inventory.Devices[0].Info != info {
		t.Fatalf("identified metadata was not preserved: %#v", got.inventory.Devices[0])
	}
	if got.inventory.Devices[1].Info != nil {
		t.Fatalf("metadata leaked to a different serial port: %#v", got.inventory.Devices[1])
	}
}
