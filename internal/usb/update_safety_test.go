package usb

import (
	"archive/zip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/micros/microsctl/internal/micropython"
)

func updateTestTarget() Target {
	return Target{Image: Image{Path: "frameworks/esp32c3/firmware.bin", Version: "3.6.0-0", MicroPython: "1.28.0"}}
}

func TestUpdateRejectsWrongOrUnknownRuntimeBeforeWriting(t *testing.T) {
	for _, machine := range []string{"ESP32-S3 module with ESP32S3", "RP2040", "unidentified"} {
		t.Run(machine, func(t *testing.T) {
			session := &fakeREPL{files: map[string][]byte{"/config/node_config.json": []byte(`{"version":"3.6.0-0"}`)}, runtime: micropython.RuntimeInfo{Machine: machine, MicroPython: "1.28.0"}}
			m := ReleaseManager{Assets: installTestFS(), OpenREPL: func(context.Context, string, REPLConfig) (replSession, error) { return session, nil }, OpenFlasher: func(string, InstallConfig) (deviceFlasher, error) {
				t.Fatal("must not flash wrong chip")
				return nil, nil
			}}
			target := updateTestTarget()
			target.Device.Info = &DeviceInfo{Chip: "ESP32-C3"} // stale cached metadata must not win
			_, err := m.Update(context.Background(), target)
			if err == nil || !strings.Contains(err.Error(), "firmware expects") || len(session.writes) != 0 || !session.reset || !session.closed {
				t.Fatalf("unsafe mismatch handling: err=%v session=%+v", err, session)
			}
		})
	}
}

func TestUpdateRuntimeUpgradePreservesFilesystem(t *testing.T) {
	before := &fakeREPL{runtime: micropython.RuntimeInfo{Machine: "ESP32C3", MicroPython: "1.27.0"}, files: map[string][]byte{
		"/config/node_config.json": []byte(`{"version":"3.6.0-0","guimeta":"...","cstmpmap":"custom"}`),
		"/config/.guimeta.key":     []byte("custom widgets"),
		"/modules/LM_custom.py":    []byte("def run(): pass"),
		"/modules/IO_custom.py":    []byte("led=7"),
		"/data/private.bin":        {0, 1, 2, 255},
	}}
	after := &fakeREPL{}
	flasher := &fakeDeviceFlasher{chip: "ESP32-C3"}
	opens := 0
	m := ReleaseManager{Assets: installTestFS(), BackupDir: t.TempDir(), OpenREPL: func(context.Context, string, REPLConfig) (replSession, error) {
		opens++
		if opens == 1 {
			return before, nil
		}
		return after, nil
	}}
	// Inspect the durable archive at the moment the erase-capable flasher opens.
	m.OpenFlasher = func(string, InstallConfig) (deviceFlasher, error) {
		archives, err := filepath.Glob(filepath.Join(m.BackupDir, "*-filesystem.zip"))
		if err != nil || len(archives) != 1 {
			t.Fatalf("filesystem not archived before flash: %v %v", archives, err)
		}
		reader, err := zip.OpenReader(archives[0])
		if err != nil {
			t.Fatal(err)
		}
		defer reader.Close()
		got := make(map[string][]byte)
		for _, entry := range reader.File {
			r, err := entry.Open()
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				t.Fatal(err)
			}
			got["/"+entry.Name] = data
		}
		if !reflect.DeepEqual(got, before.files) {
			t.Fatal("archive does not contain every original file")
		}
		return flasher, nil
	}
	_, err := m.Update(context.Background(), updateTestTarget())
	if err != nil {
		t.Fatal(err)
	}
	if !flasher.erased || opens != 2 {
		t.Fatal("MicroPython mismatch did not replace firmware")
	}
	for name, data := range before.files {
		if name == "/config/node_config.json" {
			continue
		}
		if !reflect.DeepEqual(data, after.writes[name]) {
			t.Errorf("lost %s", name)
		}
	}
}

func TestUpdateBackupFailuresResumeWithoutErasing(t *testing.T) {
	for _, failure := range []string{"directory", "inventory", "read", "runtime", "missing-directory"} {
		t.Run(failure, func(t *testing.T) {
			session := &fakeREPL{files: map[string][]byte{"/config/node_config.json": []byte(`{"version":"3.5.0-0"}`), "/config/.guimeta.key": []byte("widgets")}}
			dir := t.TempDir()
			switch failure {
			case "directory":
				block := filepath.Join(dir, "file")
				if err := os.WriteFile(block, []byte("x"), 0600); err != nil {
					t.Fatal(err)
				}
				dir = filepath.Join(block, "backups")
			case "inventory":
				session.listError = errors.New("listing failed")
			case "read":
				session.readErrors = map[string]error{"/config/.guimeta.key": errors.New("read failed")}
			case "runtime":
				session.runtimeError = errors.New("runtime read failed")
			case "missing-directory":
				dir = ""
			}
			m := ReleaseManager{Assets: installTestFS(), BackupDir: dir, OpenREPL: func(context.Context, string, REPLConfig) (replSession, error) { return session, nil }, OpenFlasher: func(string, InstallConfig) (deviceFlasher, error) {
				t.Fatal("must not erase without complete backup")
				return nil, nil
			}}
			_, err := m.Update(context.Background(), updateTestTarget())
			if err == nil || !session.reset || !session.closed || len(session.writes) != 0 {
				t.Fatalf("unsafe backup failure: %v %+v", err, session)
			}
		})
	}
}

func TestRuntimeChip(t *testing.T) {
	for machine, want := range map[string]string{
		"ESP32-C6 board with ESP32C6 [micrOS]": "esp32c6",
		"ESP32S3 module with ESP32S3":          "esp32s3",
		"TinyPICO with ESP32":                  "esp32",
		"ESP32 board with ESP32-C61":           "esp32c61",
		"ESP32-P4-rev1":                        "esp32p4rev1",
		"ESP8266 module with ESP8266":          "esp8266",
		"unknown":                              "",
	} {
		if got := runtimeChip(machine); got != want {
			t.Errorf("%q: %q != %q", machine, got, want)
		}
	}
}

func TestRestoreRetainsEmptyDirectoriesAndReplacesOnlyReleaseDestinations(t *testing.T) {
	session := &fakeREPL{}
	snapshot := []savedFile{
		{FileInfo: micropython.FileInfo{Path: "/data/empty", Directory: true}},
		{FileInfo: micropython.FileInfo{Path: "/config/.guimeta.key"}, data: []byte("widgets")},
		{FileInfo: micropython.FileInfo{Path: "/modules/LM_system.mpy"}, data: []byte("old release")},
		{FileInfo: micropython.FileInfo{Path: "/modules/LM_custom.py"}, data: []byte("custom")},
		{FileInfo: micropython.FileInfo{Path: "/node_config.json"}, data: []byte("old config")},
	}
	resources := []preparedResource{{target: "/modules/LM_system.mpy", data: []byte("new release")}}
	if err := restoreDeviceState(context.Background(), session, snapshot, resources, REPLConfig{}.withDefaults()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(session.directories, []string{"/data/empty"}) {
		t.Fatal("empty directory lost")
	}
	if len(session.writes) != 2 || string(session.writes["/config/.guimeta.key"]) != "widgets" || string(session.writes["/modules/LM_custom.py"]) != "custom" {
		t.Fatalf("incorrect restore plan: %v", session.writeOrder)
	}
}

type inventoryREPL struct {
	*fakeREPL
	entries []micropython.FileInfo
}

func (s inventoryREPL) ListFiles(context.Context) ([]micropython.FileInfo, error) {
	return s.entries, nil
}

func TestSnapshotRejectsOversizedAndUnsafeEntries(t *testing.T) {
	for _, entry := range []micropython.FileInfo{{Path: "/data/huge", Size: 1<<20 + 1}, {Path: "/../escape", Size: 0}, {Path: "/negative", Size: -1}} {
		if _, err := snapshotDevice(context.Background(), inventoryREPL{fakeREPL: &fakeREPL{}, entries: []micropython.FileInfo{entry}}); err == nil {
			t.Fatalf("accepted unsafe backup entry %+v", entry)
		}
	}
}
