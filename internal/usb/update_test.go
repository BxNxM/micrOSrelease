package usb

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/micros/microsctl/internal/micropython"
)

type fakeREPL struct {
	files        map[string][]byte
	readErrors   map[string]error
	writes       map[string][]byte
	writeOrder   []string
	writeError   error
	reset        bool
	closed       bool
	runtime      micropython.RuntimeInfo
	runtimeError error
	listError    error
	directories  []string
}

func (session *fakeREPL) Runtime(context.Context) (micropython.RuntimeInfo, error) {
	if session.runtimeError != nil {
		return micropython.RuntimeInfo{}, session.runtimeError
	}
	if session.runtime.Machine == "" {
		return micropython.RuntimeInfo{Machine: "ESP32C3 module with ESP32C3", MicroPython: "1.28.0"}, nil
	}
	return session.runtime, nil
}
func (session *fakeREPL) ListFiles(context.Context) ([]micropython.FileInfo, error) {
	if session.listError != nil {
		return nil, session.listError
	}
	var entries []micropython.FileInfo
	for name, data := range session.files {
		entries = append(entries, micropython.FileInfo{Path: name, Size: int64(len(data))})
	}
	return entries, nil
}
func (session *fakeREPL) MkdirAll(_ context.Context, name string) error {
	session.directories = append(session.directories, name)
	return nil
}

func (session *fakeREPL) ReadFile(_ context.Context, name string) ([]byte, error) {
	if err := session.readErrors[name]; err != nil {
		return nil, err
	}
	data, ok := session.files[name]
	if !ok {
		return nil, errors.New("not found")
	}
	return append([]byte(nil), data...), nil
}
func (session *fakeREPL) WriteFileAtomic(_ context.Context, name string, data []byte) error {
	if session.writeError != nil {
		return session.writeError
	}
	if session.writes == nil {
		session.writes = make(map[string][]byte)
	}
	session.writes[name] = append([]byte(nil), data...)
	session.writeOrder = append(session.writeOrder, name)
	return nil
}
func (session *fakeREPL) Reset() error { session.reset = true; return nil }
func (session *fakeREPL) Close() error { session.closed = true; return nil }

func TestUpdateBacksUpFlashesAndRestores(t *testing.T) {
	nodeConfig := []byte(`{"devfid":"kitchen/node","version":"3.5.0-0","appwd":"secret"}`)
	before := &fakeREPL{files: map[string][]byte{"/config/node_config.json": nodeConfig}}
	after := &fakeREPL{}
	flasher := &fakeDeviceFlasher{chip: "ESP32-C3"}
	openCount := 0
	files := installTestFS()
	files["modules/modules/feature.mpy"] = &fstest.MapFile{Data: []byte("module")}
	files["modules/web/assets/app.js"] = &fstest.MapFile{Data: []byte("script")}
	files["modules/data/device/defaults.json"] = &fstest.MapFile{Data: []byte("{}")}
	manager := ReleaseManager{
		Assets:    files,
		BackupDir: t.TempDir(),
		OpenFlasher: func(string, InstallConfig) (deviceFlasher, error) {
			if !before.closed {
				t.Fatal("REPL must close before opening the flasher")
			}
			return flasher, nil
		},
		OpenREPL: func(context.Context, string, REPLConfig) (replSession, error) {
			openCount++
			if openCount == 1 {
				return before, nil
			}
			if !flasher.closed {
				t.Fatal("flasher must close before reconnecting to REPL")
			}
			return after, nil
		},
	}
	var snapshots [][]Stage
	result, err := manager.Update(context.Background(), Target{
		Device: Device{Port: "/dev/ttyACM0"},
		Image: Image{
			Path: "frameworks/esp32c3/firmware.bin", Board: "esp32c3", Version: "3.6.0-0", MicroPython: "1.28.0",
		},
	}, func(stages []Stage) { snapshots = append(snapshots, stages) })
	if err != nil {
		t.Fatal(err)
	}
	if result.Operation != "update" || !strings.Contains(result.Summary, "3.5.0-0 to 3.6.0-0") {
		t.Fatalf("unexpected result: %+v", result)
	}
	if !flasher.erased || !flasher.reset || !flasher.closed {
		t.Fatalf("firmware was not fully installed: %+v", flasher)
	}
	var restored map[string]any
	if err := json.Unmarshal(after.writes["/config/node_config.json"], &restored); err != nil {
		t.Fatal(err)
	}
	if restored["version"] != "3.6.0-0" || restored["appwd"] != "secret" || string(after.writes["/modules/feature.mpy"]) != "module" {
		t.Fatalf("device state was not restored: %#v", after.writes)
	}
	if !before.closed || !after.reset || !after.closed {
		t.Fatalf("REPL lifecycle is incomplete: before=%+v after=%+v", before, after)
	}
	if openCount != 2 || len(after.writeOrder) != 4 || string(after.writes["/web/assets/app.js"]) != "script" || string(after.writes["/data/device/defaults.json"]) != "{}" {
		t.Fatalf("resources must be copied once with their relative paths: opens=%d writes=%v", openCount, after.writeOrder)
	}
	if len(snapshots) < 13 {
		t.Fatalf("got %d update stage snapshots, want at least 13", len(snapshots))
	}
	backups, err := filepath.Glob(filepath.Join(manager.BackupDir, "kitchen-node-*-node_config.json"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("configuration backup missing: %v, %v", backups, err)
	}
	backup, err := os.ReadFile(backups[0])
	if err != nil || string(backup) != string(nodeConfig) {
		t.Fatalf("invalid configuration backup: %q, %v", backup, err)
	}
}

func TestUpdateDoesNotFlashWithoutConfiguration(t *testing.T) {
	for _, data := range []string{"", "null", "[]", "invalid"} {
		t.Run("configuration="+data, func(t *testing.T) {
			flasher := &fakeDeviceFlasher{chip: "ESP32-C3"}
			session := &fakeREPL{files: map[string][]byte{"/config/node_config.json": []byte(data)}}
			manager := ReleaseManager{
				Assets:      installTestFS(),
				OpenFlasher: func(string, InstallConfig) (deviceFlasher, error) { return flasher, nil },
				OpenREPL:    func(context.Context, string, REPLConfig) (replSession, error) { return session, nil },
			}
			_, err := manager.Update(context.Background(), Target{Image: Image{Path: "frameworks/esp32c3/firmware.bin", Version: "3.6.0-0", MicroPython: "1.28.0"}})
			if err == nil || flasher.erased || !session.closed {
				t.Fatalf("invalid backup must stop before flashing and close REPL: err=%v flasher=%+v", err, flasher)
			}
		})
	}
}

func TestUpdateSkipsAlreadyCurrentFirmware(t *testing.T) {
	flasherOpened := false
	files := installTestFS()
	files["modules/modules/feature.mpy"] = &fstest.MapFile{Data: []byte("module")}
	files["modules/web/index.html"] = &fstest.MapFile{Data: []byte("page")}
	session := &fakeREPL{files: map[string][]byte{
		"/config/node_config.json": []byte(`{"version":"3.6.0-0","appwd":"secret"}`),
	}}
	manager := ReleaseManager{
		Assets: files,
		OpenFlasher: func(string, InstallConfig) (deviceFlasher, error) {
			flasherOpened = true
			return &fakeDeviceFlasher{chip: "ESP32-C3"}, nil
		},
		OpenREPL: func(context.Context, string, REPLConfig) (replSession, error) {
			return session, nil
		},
	}
	var final []Stage
	result, err := manager.Update(context.Background(), Target{Image: Image{
		Path: "frameworks/esp32c3/firmware.bin", Version: "3.6.0-0", MicroPython: "1.28.0",
	}}, func(stages []Stage) { final = stages })
	if err != nil {
		t.Fatal(err)
	}
	if flasherOpened || !strings.Contains(result.Summary, "already installed") {
		t.Fatalf("current firmware should be skipped: opened=%v result=%+v", flasherOpened, result)
	}
	for _, stage := range final[2:4] {
		if stage.State != StageSkipped {
			t.Fatalf("expected skipped flash/reconnect stages: %+v", final)
		}
	}
	if !session.reset || !session.closed || len(session.writeOrder) != 2 || string(session.writes["/modules/feature.mpy"]) != "module" || string(session.writes["/web/index.html"]) != "page" {
		t.Fatalf("same-version update must refresh resources and resume micrOS: %+v", session)
	}
	if _, wroteConfig := session.writes["/config/node_config.json"]; wroteConfig {
		t.Fatal("same-version resource refresh must preserve existing configuration")
	}
}
