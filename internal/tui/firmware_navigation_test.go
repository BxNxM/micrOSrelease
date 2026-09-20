package tui

import (
	"strings"
	"testing"

	"github.com/micros/microsctl/internal/usb"
)

func TestFirmwareBrowsingPreservesDiscoverySelection(t *testing.T) {
	for _, exitKey := range []string{"esc", "enter"} {
		t.Run(exitKey, func(t *testing.T) {
			m := model{inventory: usb.Inventory{
				Devices: []usb.Device{{Port: "test", Info: &usb.DeviceInfo{Chip: "ESP32"}}},
				Images: []usb.Image{
					{Board: "esp32", Name: "discovered-old.bin", Version: "3.9.0"},
					{Board: "esp32", Name: "discovered-new.bin", Version: "3.10.0"},
					{Board: "esp32c3", Name: "c3.bin", Version: "3.10.0"},
					{Board: "esp32c6", Name: "c6-old.bin", Version: "3.9.0"},
					{Board: "esp32c6", Name: "c6-new.bin", Version: "3.10.0"},
				},
			}}
			if !m.selectDeviceBoard() || m.imageIndex != 1 {
				t.Fatal("discovery did not select the newest matching image")
			}
			for _, key := range []string{"f", "down", "right", "right", "down"} {
				next, _ := m.handleKey(key)
				m = next.(model)
				if !m.firmwareSelected || m.imageIndex != 1 || m.selectedTarget().Image.Name != "discovered-new.bin" {
					t.Fatalf("%s replaced the discovery-selected image while browsing", key)
				}
			}
			if m.boardType != "esp32c6" || m.firmwareIndex != 3 || !strings.Contains(m.View().Content, "c6-old.bin") {
				t.Fatal("board navigation did not focus and preview a matching image")
			}
			next, _ := m.handleKey(exitKey)
			m = next.(model)
			wantBoard, wantIndex := "esp32", 1
			if exitKey == "enter" {
				wantBoard, wantIndex = "esp32c6", 3
			}
			if m.showFirmware || !m.firmwareSelected || m.imageIndex != wantIndex || m.firmwareIndex != wantIndex || m.boardType != wantBoard {
				t.Fatalf("%s left mismatched selection: board=%s image=%d focus=%d", exitKey, m.boardType, m.imageIndex, m.firmwareIndex)
			}
		})
	}
}

func TestFirmwarePickerContinuesRequestedAction(t *testing.T) {
	for _, requested := range []operation{operationInstall, operationUpdate} {
		t.Run(string(requested), func(t *testing.T) {
			m := model{inventory: usb.Inventory{
				Devices: []usb.Device{{Port: "demo"}},
				Images:  []usb.Image{{Name: "first"}, {Name: "second"}},
			}}
			next, _ := m.confirm(requested)
			m = next.(model)
			if !m.showFirmware || m.pendingOperation != requested || m.confirming {
				t.Fatal("missing firmware should open the picker and defer the action")
			}
			m.firmwareIndex = 1
			next, _ = m.firmwareKey("enter")
			m = next.(model)
			if m.showFirmware || !m.firmwareSelected || m.imageIndex != 1 || !m.confirming || m.operation != requested || m.pendingOperation != "" {
				t.Fatal("selection should return to the requested action's confirmation")
			}
		})
	}
}

func TestFirmwarePickerCancelClearsPendingAction(t *testing.T) {
	m := model{showFirmware: true, pendingOperation: operationInstall,
		inventory: usb.Inventory{Images: []usb.Image{{Name: "firmware"}}}}
	next, _ := m.firmwareKey("esc")
	m = next.(model)
	if m.showFirmware || m.pendingOperation != "" || m.firmwareSelected || m.confirming {
		t.Fatal("cancel should discard the pending action without selecting firmware")
	}
	next, _ = m.handleKey("f")
	m = next.(model)
	next, _ = m.firmwareKey("enter")
	m = next.(model)
	if !m.firmwareSelected || m.confirming || m.running {
		t.Fatal("manual selection after cancellation must not resume an action")
	}
}

func TestFirmwarePickerNavigatesNewestFirst(t *testing.T) {
	m := model{showFirmware: true, boardType: "esp32", inventory: usb.Inventory{
		Images: []usb.Image{
			{Board: "esp32", Version: "3.6.2-0"},
			{Board: "esp32c3", Version: "4.0.0-0"},
			{Board: "esp32", Version: "3.6.3-0"},
		},
	}}
	m.prepareFirmwarePicker()
	if m.firmwareIndex != 2 {
		t.Fatalf("initial firmware index = %d, want newest image at 2", m.firmwareIndex)
	}
	next, _ := m.firmwareKey("down")
	m = next.(model)
	if m.firmwareIndex != 0 {
		t.Fatalf("down selected index %d, want older image at 0", m.firmwareIndex)
	}
	next, _ = m.firmwareKey("up")
	m = next.(model)
	next, _ = m.firmwareKey("enter")
	m = next.(model)
	if !m.firmwareSelected || m.imageIndex != 2 {
		t.Fatalf("selection = %v, image index = %d, want newest image at 2", m.firmwareSelected, m.imageIndex)
	}
}
