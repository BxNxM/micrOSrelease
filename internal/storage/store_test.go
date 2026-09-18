package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/micros/microsctl/internal/network"
)

func TestPersistentDevices(t *testing.T) {
	root := t.TempDir()
	store, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(store.FirmwareDir()); err != nil {
		t.Fatal(err)
	}
	if nodes, err := store.LoadDevices(); err != nil || len(nodes) != 0 {
		t.Fatalf("empty cache: %v %v", nodes, err)
	}
	want := network.Device{UID: "test-uid", Name: "Kitchen", Address: "10.0.1.3:9008", Online: true, Features: map[string]string{"webui": "ON"}}
	if err := store.SaveDevices([]network.Device{want}); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	service := network.Service{Store: reopened}
	nodes, err := service.Cached()
	if err != nil || len(nodes) != 1 {
		t.Fatalf("reload: %v %v", nodes, err)
	}
	if nodes[0].UID != want.UID || !nodes[0].Cached || nodes[0].Features["webui"] != "ON" {
		t.Fatalf("bad cached node: %+v", nodes[0])
	}
	if err := os.WriteFile(filepath.Join(root, "devices.json"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Cached(); err == nil {
		t.Fatal("corrupt cache must be reported")
	}
}
