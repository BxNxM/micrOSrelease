package usb

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	assets "github.com/micros/microsctl/storage"
)

func hardwareInstallTarget(t *testing.T) (ReleaseManager, Target, InstallConfig) {
	t.Helper()
	port := os.Getenv("MICROS_TEST_INSTALL_PORT")
	if port == "" {
		t.Skip("set MICROS_TEST_INSTALL_PORT for live install testing")
	}
	board := os.Getenv("MICROS_TEST_INSTALL_BOARD")
	manager := ReleaseManager{Assets: assets.Files(), BackupDir: os.Getenv("MICROS_TEST_BACKUP_DIR")}
	images, err := Images(manager.Assets)
	if err != nil {
		t.Fatal(err)
	}
	index, ok := LatestBoardImage(images, board)
	if !ok {
		t.Fatal("set MICROS_TEST_INSTALL_BOARD to the selected board")
	}
	target := Target{Device: Device{Port: port}, Image: images[index]}
	config, _, err := loadInstallConfig(manager.Assets, target.Image)
	if err != nil {
		t.Fatal(err)
	}
	return manager, target, config
}

// Connection only: exercises the install bootloader/stub without erasing flash.
func TestHardwareUSBInstallConnection(t *testing.T) {
	_, target, config := hardwareInstallTarget(t)
	t.Logf("Connect using install settings: %s, %s, reset=%s", target.Device.Port, config.Chip, config.ResetMode)
	f, err := newESPDeviceFlasher(target.Device, config)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	defer f.Reset()
	if normalizeChip(f.ChipName()) != normalizeChip(config.Chip) {
		t.Fatalf("unexpected chip %s", f.ChipName())
	}
	_, id, err := f.FlashID()
	if err != nil {
		t.Fatal(err)
	}
	if flashSizeFromID(id) == "" {
		t.Fatalf("invalid flash ID %04X", id)
	}
	t.Logf("Connected to %s, flash %s", f.ChipName(), flashSizeFromID(id))
}

// Opt-in destructive install test. Save a durable backup first, verify installed
// resources, then restore user files/configuration while retaining new resources.
func TestHardwareUSBInstall(t *testing.T) {
	manager, target, config := hardwareInstallTarget(t)
	if manager.BackupDir == "" {
		t.Fatal("MICROS_TEST_BACKUP_DIR is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	before, err := openMicroPythonREPL(ctx, target.Device.Port, config.REPL)
	if err != nil {
		t.Fatal(err)
	}
	defer before.Close()
	info, err := before.Runtime(ctx)
	if err != nil || runtimeChip(info.Machine) != normalizeChip(config.Chip) {
		_ = before.Reset()
		t.Fatalf("pre-install runtime validation: %s, %v", info.Machine, err)
	}
	t.Log("Saving verified pre-install filesystem backup")
	snapshot, backupErr := snapshotDevice(ctx, before)
	var backupPath string
	if backupErr == nil {
		backupPath, backupErr = archiveDeviceState(manager.BackupDir, "install-test", snapshot)
	}
	resetErr, closeErr := before.Reset(), before.Close()
	if backupErr != nil {
		t.Fatal(backupErr)
	}
	t.Logf("Backup: %s (%d entries)", backupPath, len(snapshot))
	if resetErr != nil || closeErr != nil {
		t.Fatalf("reset=%v close=%v", resetErr, closeErr)
	}
	last := ""
	result, err := manager.Install(ctx, target, func(stages []Stage) {
		for _, stage := range stages {
			message := stage.Name + " · " + stage.Detail
			if stage.State == StageRunning && message != last {
				last = message
				t.Log(message)
			}
		}
	})
	if err != nil {
		t.Fatalf("install: %v (backup: %s)", err, backupPath)
	}
	t.Log(result.Summary)
	after, _, err := manager.reconnectREPL(ctx, result.Target, config, nil)
	if err != nil {
		t.Fatalf("post-install reconnect: %v (backup: %s)", err, backupPath)
	}
	defer after.Close()
	defer after.Reset()
	resources, err := prepareResources(manager.Assets, config)
	if err != nil {
		t.Fatal(err)
	}
	for _, resource := range resources {
		got, err := after.ReadFile(ctx, resource.target)
		if err != nil || !bytes.Equal(got, resource.data) {
			t.Errorf("installed resource verification failed: %s (%v)", resource.target, err)
		}
	}
	t.Logf("Verified %d installed resources; restoring saved user files/configuration", len(resources))
	if err := restoreDeviceState(ctx, after, snapshot, resources, config.REPL); err != nil {
		t.Fatal(err)
	}
	for _, entry := range snapshot {
		for _, name := range config.REPL.ConfigPaths {
			if entry.Path == name {
				if err := after.WriteFileAtomic(ctx, entry.Path, entry.data); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	t.Log("Install verified; user files/configuration restored")
}
