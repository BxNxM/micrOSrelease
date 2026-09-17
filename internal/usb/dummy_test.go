package usb

import (
	"context"
	"testing"
	"testing/fstest"
	"time"
)

func TestDummyManagerWorkflow(t *testing.T) {
	manager := DummyManager{Delay: time.Millisecond, Assets: fstest.MapFS{
		"frameworks/micrOS-esp32-1.28.0-3.6.0-0.bin": &fstest.MapFile{Data: []byte("demo")},
		"frameworks/README.md":                       &fstest.MapFile{Data: []byte("ignored")},
	}}
	inventory, err := manager.Inventory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Devices) == 0 || len(inventory.Images) == 0 {
		t.Fatal("dummy inventory must contain selectable targets")
	}
	target := Target{Device: inventory.Devices[0], Image: inventory.Images[0]}
	if len(inventory.Images) != 1 || target.Image.Version != "3.6.0-0" || target.Image.MicroPython != "1.28.0" || target.Image.Size != 4 {
		t.Fatalf("unexpected firmware catalog: %+v", inventory.Images)
	}
	result, err := manager.Install(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	if result.Operation != "install" || len(result.Log) == 0 {
		t.Fatalf("unexpected install result: %#v", result)
	}
}
