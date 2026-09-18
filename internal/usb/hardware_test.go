package usb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/micros/microsctl/internal/micropython"
	assets "github.com/micros/microsctl/storage"
)

// Opt-in only: this test resets and updates a connected device. Backups are kept
// in MICROS_TEST_BACKUP_DIR, outside Go's automatically deleted test directories.
// MICROS_TEST_FLASH=1 additionally exercises full erase/restore with the matching
// bundled image even if the installed runtime is current.
func TestHardwareUSBUpdate(t *testing.T) {
	port := os.Getenv("MICROS_TEST_PORT")
	if port == "" {
		t.Skip("set MICROS_TEST_PORT and MICROS_TEST_BACKUP_DIR for hardware testing")
	}
	backupDir := os.Getenv("MICROS_TEST_BACKUP_DIR")
	if backupDir == "" {
		t.Fatal("MICROS_TEST_BACKUP_DIR is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	manager := ReleaseManager{Assets: assets.Files(), BackupDir: backupDir}
	session, err := openMicroPythonREPL(ctx, port, (REPLConfig{}).withDefaults())
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	info, err := session.Runtime(ctx)
	if err != nil {
		_ = session.Reset()
		t.Fatal(err)
	}
	t.Logf("Runtime: %s, MicroPython %s", info.Machine, info.MicroPython)
	images, err := Images(manager.Assets)
	if err != nil {
		t.Fatal(err)
	}
	var selected Image
	for _, image := range images {
		cfg, _, err := loadInstallConfig(manager.Assets, image)
		if err == nil && normalizeChip(cfg.Chip) == runtimeChip(info.Machine) {
			selected = image
			break
		}
	}
	if selected.Path == "" {
		t.Fatal("no matching bundled image")
	}
	config, _, err := loadInstallConfig(manager.Assets, selected)
	if err != nil {
		t.Fatal(err)
	}
	before, backupErr := snapshotDevice(ctx, session)
	if backupErr == nil {
		var name string
		name, backupErr = archiveDeviceState(backupDir, "hardware-test", before)
		t.Logf("Pre-test filesystem backup: %s (%d entries)", name, len(before))
	}
	resetErr := session.Reset()
	closeErr := session.Close()
	if backupErr != nil {
		t.Fatal(backupErr)
	}
	if resetErr != nil || closeErr != nil {
		t.Fatalf("reset=%v close=%v", resetErr, closeErr)
	}
	device, err := manager.Probe(ctx, Device{Port: port})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Identified %s, flash %s", device.Info.Chip, device.Info.FlashSize)
	// Allow normal application startup, including its watchdog, before updating.
	time.Sleep(10 * time.Second)
	if os.Getenv("MICROS_TEST_FLASH") == "1" {
		firstOpen := true
		manager.OpenREPL = func(ctx context.Context, port string, cfg REPLConfig) (replSession, error) {
			live, err := openMicroPythonREPL(ctx, port, cfg)
			if err != nil {
				return nil, err
			}
			if firstOpen {
				firstOpen = false
				return forceHardwareFlash{live}, nil
			}
			return live, nil
		}
	}
	last := ""
	result, err := manager.Update(ctx, Target{Device: device, Image: selected}, func(stages []Stage) {
		for _, stage := range stages {
			if stage.State == StageRunning && last != stage.Name {
				last = stage.Name
				t.Log(stage.Name)
			}
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(result.Summary)
	time.Sleep(10 * time.Second)
	after, _, err := manager.reconnectREPL(ctx, result.Target, config, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer after.Close()
	defer after.Reset()
	resources, err := prepareResources(manager.Assets, config)
	if err != nil {
		t.Fatal(err)
	}
	replaced := make(map[string]bool)
	for _, resource := range resources {
		replaced[resource.target] = true
	}
	for _, entry := range before {
		if entry.Directory || replaced[entry.Path] || strings.HasPrefix(entry.Path, "/logs/") {
			continue
		}
		got, err := after.ReadFile(ctx, entry.Path)
		if err != nil {
			t.Fatalf("read preserved %s: %v", entry.Path, err)
		}
		if strings.HasSuffix(entry.Path, "node_config.json") {
			var oldSettings, newSettings map[string]any
			if json.Unmarshal(entry.data, &oldSettings) != nil || json.Unmarshal(got, &newSettings) != nil {
				t.Fatal("invalid config")
			}
			// Runtime metadata may change on startup; all user settings must survive.
			for _, key := range []string{"version", "devip", "hwuid"} {
				delete(oldSettings, key)
				delete(newSettings, key)
			}
			oldJSON, _ := json.Marshal(oldSettings)
			newJSON, _ := json.Marshal(newSettings)
			if !bytes.Equal(oldJSON, newJSON) {
				t.Fatal("user configuration changed")
			}
		} else if !bytes.Equal(got, entry.data) {
			t.Errorf("preserved content changed: %s", entry.Path)
		}
	}
	for _, resource := range resources {
		got, err := after.ReadFile(ctx, resource.target)
		if err != nil || !bytes.Equal(got, resource.data) {
			t.Fatalf("resource verification failed: %s (%v)", resource.target, err)
		}
	}
	t.Logf("Verified configuration, preserved files, and %d release resources", len(resources))
}

type forceHardwareFlash struct{ replSession }

func (s forceHardwareFlash) Runtime(ctx context.Context) (micropython.RuntimeInfo, error) {
	info, err := s.replSession.Runtime(ctx)
	// Exercise the runtime-upgrade branch without modifying on-device metadata.
	info.MicroPython = "hardware-test-force-flash"
	return info, err
}

// This exercises the same reconnect stage without erasing flash or writing files.
// After READY, physically unplug USB for at least two seconds, then reconnect it.
func TestHardwareUSBReconnect(t *testing.T) {
	port := os.Getenv("MICROS_TEST_RECONNECT_PORT")
	if port == "" {
		t.Skip("set MICROS_TEST_RECONNECT_PORT for a live unplug/replug test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	manager := ReleaseManager{}
	device := manager.reconnectDevice(ctx, Device{Port: port})
	chip := os.Getenv("MICROS_TEST_RECONNECT_CHIP")
	if !supportedESPChip(chip) {
		t.Fatal("set MICROS_TEST_RECONNECT_CHIP to the connected chip type")
	}
	config := InstallConfig{Chip: chip, REPL: (REPLConfig{}).withDefaults()}
	target := Target{Device: device}
	detached := false
	manager.Discover = func(ctx context.Context) ([]Device, error) {
		devices, err := discoverDevices(ctx)
		if err == nil && len(reconnectCandidates(device, devices)) == 0 {
			detached = true
		}
		return devices, err
	}
	manager.OpenREPL = func(ctx context.Context, port string, cfg REPLConfig) (replSession, error) {
		if !detached {
			return nil, fmt.Errorf("waiting for physical unplug/replug")
		}
		return openReconnectREPL(ctx, port, cfg)
	}
	t.Logf("READY: unplug/replug %s; stable USB identity available: %v", device.Port, device.USBSerial != "")
	session, connected, err := manager.reconnectREPL(ctx, target, config, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if err := session.Reset(); err != nil {
		t.Fatal(err)
	}
	t.Logf("PASS: reconnected on %s after physical detach; runtime validated and reset acknowledged", connected.Port)
}
