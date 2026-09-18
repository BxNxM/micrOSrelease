package usb

import (
	"context"
	"testing"
	"testing/fstest"
)

func TestReleaseManagerInventory(t *testing.T) {
	manager := ReleaseManager{Assets: fstest.MapFS{
		"frameworks/esp32/micrOS-esp32-1.28.0-3.6.0-0.bin": &fstest.MapFile{Data: []byte("demo")},
		"frameworks/esp32/install.json": &fstest.MapFile{Data: []byte(`{
            "version": 1, "protocol": "esp-rom", "chip": "esp32",
            "initial_baud": 115200, "flash_baud": 460800, "flash_offset": "0x1000",
            "erase_flash": true, "compress": true, "reset_mode": "auto", "reset_after_flash": true
        }`)},
		"frameworks/README.md": &fstest.MapFile{Data: []byte("ignored")},
	}, Discover: func(context.Context) ([]Device, error) {
		return []Device{{Port: "/dev/ttyUSB0"}}, nil
	}}
	inventory, err := manager.Inventory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Devices) == 0 || len(inventory.Images) == 0 {
		t.Fatal("inventory must contain selectable targets")
	}
	target := Target{Device: inventory.Devices[0], Image: inventory.Images[0]}
	if len(inventory.Images) != 1 || target.Image.Version != "3.6.0-0" || target.Image.MicroPython != "1.28.0" || target.Image.Size != 4 {
		t.Fatalf("unexpected firmware catalog: %+v", inventory.Images)
	}
}
