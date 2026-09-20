package tui

import (
	"github.com/micros/microsctl/internal/usb"
	"strings"
	"testing"
)

func TestOpeningActionsSelectsCurrentBoardFirmware(t *testing.T) {
	for _, delayed := range []bool{false, true} {
		m := model{showNodes: true}
		inventory := usb.Inventory{Images: []usb.Image{{Board: "esp32", Name: "firmware.bin"}}}
		if !delayed {
			m.inventory = inventory
		}
		next, _ := m.handleKey("enter")
		m = next.(model)
		if delayed {
			next, _ = m.Update(inventoryMsg{inventory: inventory})
			m = next.(model)
		}
		if m.showNodes || m.boardType != "esp32" || !m.firmwareSelected || m.imageIndex != 0 {
			t.Fatalf("opening Actions should select the single firmware (delayed inventory: %v)", delayed)
		}
	}
}

func TestSwitchBoardSelectsNewestFirmware(t *testing.T) {
	m := model{boardType: "esp32", firmwareSelected: true, inventory: usb.Inventory{
		Images: []usb.Image{{Board: "esp32"}, {Board: "esp32c3", Name: "newest-c3.bin", Version: "3.10.0"}, {Board: "esp32c3", Version: "3.9.0"}},
	}}
	m.switchBoard(1)
	if m.boardType != "esp32c3" || !m.firmwareSelected || m.imageIndex != 1 || m.firmwareIndex != 1 {
		t.Fatal("board switch should select and preview the newest matching image")
	}
	if preview := m.renderState().ReleaseTargetWidget(80); !strings.Contains(preview, "newest-c3.bin") {
		t.Fatal("USB Tools preview did not show the newly selected board firmware")
	}
	next, _ := m.firmwareKey("down")
	m = next.(model)
	if m.firmwareIndex != 2 {
		t.Fatal("should navigate within board images")
	}
	next, _ = m.firmwareKey("enter")
	m = next.(model)
	if !m.firmwareSelected || m.imageIndex != 2 {
		t.Fatal("Enter should select firmware explicitly")
	}
	m.switchBoard(0)
	if !m.firmwareSelected || m.imageIndex != 2 || m.firmwareIndex != 2 {
		t.Fatal("reopening USB Tools should retain the user's older firmware choice")
	}
	m.switchBoard(1)
	if m.boardType != "esp32" || !m.firmwareSelected || m.imageIndex != 0 {
		t.Fatal("board navigation should wrap and auto-select the only binary")
	}
}

func TestSwitchSingleBoardSelectsOnlyBinary(t *testing.T) {
	m := model{boardType: "esp32", inventory: usb.Inventory{
		Images: []usb.Image{{Board: "esp32", Name: "firmware.bin"}},
	}}
	m.switchBoard(1)
	if !m.firmwareSelected || m.imageIndex != 0 {
		t.Fatal("explicit board navigation should select its only binary")
	}
}
