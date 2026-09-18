package usb

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/micros/microsctl/internal/micropython"
)

func TestReconnectCandidatesFollowIdentityWithoutGuessing(t *testing.T) {
	target := Device{Port: "old", USBSerial: "board-a", USBVID: "303A", USBPID: "1001"}
	other := Device{Port: "old", USBSerial: "board-b", USBVID: "303A"}
	moved := Device{Port: "new", USBSerial: "board-a", USBVID: "303a", USBPID: "0002"}
	if got := reconnectCandidates(target, []Device{other, moved}); !reflect.DeepEqual(got, []Device{moved}) {
		t.Fatalf("wrong reconnect target: %+v", got)
	}
	duplicate := moved
	duplicate.Port = "another"
	if got := reconnectCandidates(target, []Device{moved, duplicate}); len(got) != 0 {
		t.Fatal("ambiguous serial numbers must not pick a board")
	}
	if got := reconnectCandidates(Device{Port: "old"}, []Device{moved}); len(got) != 0 {
		t.Fatal("unknown identity must not follow another port")
	}
	if got := reconnectCandidates(Device{Port: "/dev/tty.usbmodem1"}, []Device{{Port: "/dev/cu.usbmodem1"}}); len(got) != 1 {
		t.Fatal("macOS aliases did not match")
	}
}

func TestUpdateContinuesRestoreAfterUSBReturnsOnNewPort(t *testing.T) {
	before := &fakeREPL{files: map[string][]byte{"/config/node_config.json": []byte(`{"version":"old","devfid":"preserved"}`), "/data/custom.txt": []byte("keep me")}}
	after := &fakeREPL{}
	flasher := &fakeDeviceFlasher{chip: "ESP32-C3"}
	original := Device{Port: "old", USBSerial: "board-a", USBVID: "303A"}
	returned := original
	returned.Port = "new"
	target := updateTestTarget()
	target.Device = original
	flashes, scans := 0, 0
	manager := ReleaseManager{Assets: installTestFS(), BackupDir: t.TempDir(),
		Discover: func(context.Context) ([]Device, error) {
			scans++
			if scans == 1 {
				return nil, nil
			}
			if scans == 2 {
				return []Device{{Port: "old", USBSerial: "unrelated", USBVID: "303A"}}, nil
			}
			return []Device{returned}, nil
		},
		OpenFlasher: func(string, InstallConfig) (deviceFlasher, error) { flashes++; return flasher, nil },
		OpenREPL: func(_ context.Context, port string, _ REPLConfig) (replSession, error) {
			if !flasher.erased {
				if port != "old" {
					t.Fatal("wrong initial port")
				}
				return before, nil
			}
			if port != "new" {
				t.Fatal("reconnect opened missing or unrelated device")
			}
			return after, nil
		},
	}
	waiting := false
	result, err := manager.Update(context.Background(), target, func(stages []Stage) {
		for _, s := range stages {
			if s.State == StageRunning && s.CanReconnect {
				waiting = true
				if !strings.Contains(s.Detail, "reconnect") {
					t.Fatal("missing reconnect instructions")
				}
			}
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if !waiting || flashes != 1 || result.Target.Device.Port != "new" || string(after.writes["/data/custom.txt"]) != "keep me" || !after.reset {
		t.Fatalf("failed resumed update: waiting=%v flashes=%d port=%s", waiting, flashes, result.Target.Device.Port)
	}
}

func TestReconnectWaitCanBeCancelledAndHasNoDefaultDeadline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager := ReleaseManager{Discover: func(ctx context.Context) ([]Device, error) {
		if (REPLConfig{}).withDefaults().ReconnectTimeoutSeconds != 0 {
			t.Fatal("default reconnect unexpectedly expires")
		}
		cancel()
		return nil, nil
	}, OpenREPL: func(context.Context, string, REPLConfig) (replSession, error) {
		t.Fatal("opened absent board")
		return nil, nil
	}}
	_, _, err := manager.reconnectREPL(ctx, Target{Device: Device{USBSerial: "board"}}, InstallConfig{Chip: "esp32c3", REPL: (REPLConfig{}).withDefaults()}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func TestReconnectHonorsExplicitTimeout(t *testing.T) {
	manager := ReleaseManager{Discover: func(context.Context) ([]Device, error) { return nil, nil }}
	started := time.Now()
	_, _, err := manager.reconnectREPL(context.Background(), Target{Device: Device{USBSerial: "absent"}}, InstallConfig{Chip: "esp32c3", REPL: REPLConfig{ReconnectTimeoutSeconds: 1}}, nil)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(started) > 2*time.Second {
		t.Fatalf("timeout not honored: %v", err)
	}
}

func TestReconnectValidatesInterpreterBeforeReturning(t *testing.T) {
	wrong := &fakeREPL{runtime: micropython.RuntimeInfo{Machine: "ESP32S3", MicroPython: "1.28.0"}}
	right := &fakeREPL{}
	opens := 0
	manager := ReleaseManager{Discover: func(context.Context) ([]Device, error) { return []Device{{Port: "same"}}, nil }, OpenREPL: func(context.Context, string, REPLConfig) (replSession, error) {
		opens++
		if opens == 1 {
			return wrong, nil
		}
		return right, nil
	}}
	session, _, err := manager.reconnectREPL(context.Background(), Target{Device: Device{Port: "same"}}, InstallConfig{Chip: "esp32c3", REPL: (REPLConfig{}).withDefaults()}, nil)
	if err != nil || session != right || !wrong.reset || !wrong.closed || len(wrong.writes) != 0 {
		t.Fatalf("wrong runtime accepted: %v", err)
	}
}

func TestInstallContinuesResourceCopyAfterUSBReplug(t *testing.T) {
	files := installTestFS()
	files["modules/data/default.txt"] = &fstest.MapFile{Data: []byte("installed resource")}
	session := &fakeREPL{}
	target := updateTestTarget()
	target.Device = Device{Port: "old", USBSerial: "same-board", USBVID: "303A"}
	returned := target.Device
	returned.Port = "new"
	flashes, scans := 0, 0
	manager := ReleaseManager{Assets: files,
		Discover: func(context.Context) ([]Device, error) {
			scans++
			if scans == 1 {
				return nil, nil
			}
			return []Device{returned}, nil
		},
		OpenFlasher: func(string, InstallConfig) (deviceFlasher, error) {
			flashes++
			return &fakeDeviceFlasher{chip: "ESP32-C3"}, nil
		},
		OpenREPL: func(_ context.Context, port string, _ REPLConfig) (replSession, error) {
			if port != "new" {
				t.Fatal("opened stale port")
			}
			return session, nil
		},
	}
	result, err := manager.Install(context.Background(), target)
	if err != nil || flashes != 1 || result.Target.Device.Port != "new" || string(session.writes["/data/default.txt"]) != "installed resource" || !session.reset {
		t.Fatalf("install did not resume after replug: %v", err)
	}
}
