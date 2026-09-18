package usb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

type fakeDeviceFlasher struct {
	chip         string
	erased       bool
	flashed      []byte
	offset       uint32
	reset        bool
	closed       bool
	flashError   error
	flashID      uint16
	flashIDError error
}

func (f *fakeDeviceFlasher) ChipName() string { return f.chip }
func (f *fakeDeviceFlasher) FlashID() (uint8, uint16, error) {
	if f.flashID == 0 {
		return 0, 0x4016, f.flashIDError
	}
	return 0, f.flashID, f.flashIDError
}
func (f *fakeDeviceFlasher) EraseFlash(func(int, int)) error {
	f.erased = true
	return nil
}
func (f *fakeDeviceFlasher) FlashImage(data []byte, offset uint32, _ func(int, int)) error {
	f.flashed = append([]byte(nil), data...)
	f.offset = offset
	return f.flashError
}
func (f *fakeDeviceFlasher) Reset()       { f.reset = true }
func (f *fakeDeviceFlasher) Close() error { f.closed = true; return nil }

func TestInstallUsesBoardConfig(t *testing.T) {
	files := installTestFS()
	flasher := &fakeDeviceFlasher{chip: "ESP32-C3"}
	var openedPort string
	var openedConfig InstallConfig
	manager := ReleaseManager{Assets: files, OpenFlasher: func(port string, config InstallConfig) (deviceFlasher, error) {
		openedPort, openedConfig = port, config
		return flasher, nil
	}}
	var snapshots [][]Stage
	result, err := manager.Install(context.Background(), Target{
		Device: Device{Port: "/dev/ttyACM0"},
		Image:  Image{Path: "frameworks/esp32c3/firmware.bin", Board: "esp32c3"},
	}, func(stages []Stage) { snapshots = append(snapshots, stages) })
	if err != nil {
		t.Fatal(err)
	}
	if openedPort != "/dev/ttyACM0" || openedConfig.FlashBaud != 460800 {
		t.Fatalf("unexpected open parameters: %q %+v", openedPort, openedConfig)
	}
	if !flasher.erased || !flasher.reset || !flasher.closed || flasher.offset != 0 || !reflect.DeepEqual(flasher.flashed, []byte("firmware")) {
		t.Fatalf("unexpected flasher calls: %+v", flasher)
	}
	if result.Operation != "install" || len(snapshots) != 1+2*len(result.Log) {
		t.Fatalf("unexpected result or stage snapshots: %#v, %d", result, len(snapshots))
	}
}

func TestInstallRejectsWrongChipBeforeErase(t *testing.T) {
	flasher := &fakeDeviceFlasher{chip: "ESP32-S3"}
	manager := ReleaseManager{Assets: installTestFS(), OpenFlasher: func(string, InstallConfig) (deviceFlasher, error) {
		return flasher, nil
	}}
	_, err := manager.Install(context.Background(), Target{
		Device: Device{Port: "COM7"},
		Image:  Image{Path: "frameworks/esp32c3/firmware.bin"},
	})
	if err == nil || !strings.Contains(err.Error(), "firmware expects esp32c3") {
		t.Fatalf("expected chip mismatch, got %v", err)
	}
	if flasher.erased || !flasher.closed {
		t.Fatalf("wrong chip must be closed without erasing: %+v", flasher)
	}
}

func TestInstallStopsAfterFlashFailure(t *testing.T) {
	want := errors.New("write failed")
	flasher := &fakeDeviceFlasher{chip: "esp32c3", flashError: want}
	manager := ReleaseManager{Assets: installTestFS(), OpenFlasher: func(string, InstallConfig) (deviceFlasher, error) {
		return flasher, nil
	}}
	_, err := manager.Install(context.Background(), Target{Image: Image{Path: "frameworks/esp32c3/firmware.bin"}})
	if !errors.Is(err, want) || flasher.reset || !flasher.closed {
		t.Fatalf("unexpected failed install state: err=%v flasher=%+v", err, flasher)
	}
}

func TestInstallCopiesBundledResources(t *testing.T) {
	for _, failCopy := range []bool{false, true} {
		t.Run(fmt.Sprintf("copyFailure=%v", failCopy), func(t *testing.T) {
			files := installTestFS()
			files["modules/README.md"] = &fstest.MapFile{Data: []byte("documentation")}
			files["modules/modules/feature.mpy"] = &fstest.MapFile{Data: []byte("module")}
			files["modules/web/assets/app.js"] = &fstest.MapFile{Data: []byte("script")}
			files["modules/data/device/defaults.json"] = &fstest.MapFile{Data: []byte("{}")}
			flasher := &fakeDeviceFlasher{chip: "ESP32-C3"}
			session := &fakeREPL{}
			copyErr := errors.New("device filesystem is full")
			if failCopy {
				session.writeError = copyErr
			}
			manager := ReleaseManager{
				Assets:      files,
				OpenFlasher: func(string, InstallConfig) (deviceFlasher, error) { return flasher, nil },
				OpenREPL: func(context.Context, string, REPLConfig) (replSession, error) {
					if !flasher.reset || !flasher.closed {
						t.Fatal("firmware must boot and release serial port before copying resources")
					}
					return session, nil
				},
			}
			_, err := manager.Install(context.Background(), Target{Image: Image{Path: "frameworks/esp32c3/firmware.bin"}})
			if failCopy {
				if !errors.Is(err, copyErr) || session.reset || !session.closed {
					t.Fatalf("copy failure must stop install and close REPL: err=%v session=%+v", err, session)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			want := map[string][]byte{
				"/modules/feature.mpy":       []byte("module"),
				"/web/assets/app.js":         []byte("script"),
				"/data/device/defaults.json": []byte("{}"),
			}
			if !reflect.DeepEqual(session.writes, want) || len(session.writeOrder) != len(want) || !session.reset || !session.closed {
				t.Fatalf("install did not copy and finish correctly: %+v", session)
			}
		})
	}
}

func TestMissingResourceStopsBeforeHardwareAccess(t *testing.T) {
	files := installTestFS()
	var config InstallConfig
	if err := json.Unmarshal(files["frameworks/esp32c3/install.json"].Data, &config); err != nil {
		t.Fatal(err)
	}
	config.REPL.Resources = []UpdateResource{{Source: "modules/missing.mpy", Target: "/modules/missing.mpy"}}
	encoded, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	files["frameworks/esp32c3/install.json"].Data = encoded
	manager := ReleaseManager{
		Assets: files,
		OpenFlasher: func(string, InstallConfig) (deviceFlasher, error) {
			t.Fatal("must preflight resources before flash access")
			return nil, nil
		},
		OpenREPL: func(context.Context, string, REPLConfig) (replSession, error) {
			t.Fatal("must preflight resources before REPL access")
			return nil, nil
		},
	}
	target := Target{Image: Image{Path: "frameworks/esp32c3/firmware.bin", Version: "3.6.0-0"}}
	for _, operation := range []func(context.Context, Target, ...func([]Stage)) (Result, error){manager.Install, manager.Update} {
		if _, err := operation(context.Background(), target); err == nil || !strings.Contains(err.Error(), "modules/missing.mpy") {
			t.Fatalf("expected missing resource error before touching hardware: %v", err)
		}
	}
}

func installTestFS() fstest.MapFS {
	return fstest.MapFS{
		"frameworks/esp32c3/firmware.bin": {Data: []byte("firmware")},
		"frameworks/esp32c3/install.json": {Data: []byte(`{
            "version": 1,
            "protocol": "esp-rom",
            "chip": "esp32c3",
            "initial_baud": 115200,
            "flash_baud": 460800,
            "flash_offset": "0x0",
            "erase_flash": true,
            "compress": true,
            "reset_mode": "auto",
            "reset_after_flash": true
        }`)},
	}
}

func TestInstallRejectsInsufficientOrUnknownFlashBeforeErase(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   uint16
		err  error
	}{
		{"too-small", 0x4012, nil},
		{"unknown", 0x4001, nil},
		{"unreadable", 0, errors.New("JEDEC read failed")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := installTestFS()
			files["frameworks/esp32c3/firmware.bin"].Data = make([]byte, 1<<18+1)
			flasher := &fakeDeviceFlasher{chip: "ESP32-C3", flashID: tc.id, flashIDError: tc.err}
			manager := ReleaseManager{Assets: files, OpenFlasher: func(string, InstallConfig) (deviceFlasher, error) { return flasher, nil }}
			_, err := manager.Install(context.Background(), Target{Image: Image{Path: "frameworks/esp32c3/firmware.bin"}})
			if err == nil || flasher.erased || len(flasher.flashed) > 0 || !flasher.reset || !flasher.closed {
				t.Fatalf("unsafe capacity rejection: err=%v erased=%v reset=%v closed=%v", err, flasher.erased, flasher.reset, flasher.closed)
			}
		})
	}
}

func TestOversizedResourceStopsBeforeHardwareAccess(t *testing.T) {
	files := installTestFS()
	files["modules/data/large.bin"] = &fstest.MapFile{Data: make([]byte, 1<<20+1)}
	manager := ReleaseManager{Assets: files,
		OpenFlasher: func(string, InstallConfig) (deviceFlasher, error) {
			t.Fatal("oversized resource reached flasher")
			return nil, nil
		},
		OpenREPL: func(context.Context, string, REPLConfig) (replSession, error) {
			t.Fatal("oversized resource reached REPL")
			return nil, nil
		},
	}
	for _, operation := range []func(context.Context, Target, ...func([]Stage)) (Result, error){manager.Install, manager.Update} {
		if _, err := operation(context.Background(), updateTestTarget()); err == nil || !strings.Contains(err.Error(), "transfer limit") {
			t.Fatalf("expected transfer preflight error, got %v", err)
		}
	}
}

func TestInvalidFirmwareStopsBeforeHardwareAccess(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(fmt.Sprintf("missing=%v", missing), func(t *testing.T) {
			files := installTestFS()
			target := updateTestTarget()
			if missing {
				delete(files, target.Image.Path)
			} else {
				files[target.Image.Path].Data = nil
			}
			manager := ReleaseManager{Assets: files,
				OpenFlasher: func(string, InstallConfig) (deviceFlasher, error) {
					t.Error("invalid firmware reached flasher")
					return nil, errors.New("unexpected flasher access")
				},
				OpenREPL: func(context.Context, string, REPLConfig) (replSession, error) {
					t.Error("invalid firmware interrupted the running application")
					return nil, errors.New("unexpected REPL access")
				},
			}
			for _, operation := range []func(context.Context, Target, ...func([]Stage)) (Result, error){manager.Install, manager.Update} {
				if _, err := operation(context.Background(), target); err == nil || !strings.Contains(err.Error(), "firmware") {
					t.Fatalf("expected firmware preflight error, got %v", err)
				}
			}
		})
	}
}
