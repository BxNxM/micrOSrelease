package tui

import (
	"testing"

	"github.com/micros/microsctl/internal/usb"
)

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
