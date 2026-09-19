package usb

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/fstest"
)

type progressFlasher struct {
	fakeDeviceFlasher
	check func(string)
}

func (f *progressFlasher) EraseFlash(progress func(int, int)) error {
	f.check("Erase flash")
	return f.fakeDeviceFlasher.EraseFlash(progress)
}

func (f *progressFlasher) FlashImage(data []byte, offset uint32, progress func(int, int)) error {
	f.check("Write and verify firmware")
	progress(1, 2)
	f.check("50% transferred")
	progress(2, 2)
	f.check("100% transferred")
	return f.fakeDeviceFlasher.FlashImage(data, offset, progress)
}

func (f *progressFlasher) Reset() {
	f.check("Reset device")
	f.fakeDeviceFlasher.Reset()
}

func TestFirmwareDetailsDuringInstallAndUpdate(t *testing.T) {
	for _, install := range []bool{false, true} {
		for _, failWrite := range []bool{false, true} {
			files := installTestFS()
			files["modules/main.py"] = &fstest.MapFile{Data: []byte("startup")}
			var latest string
			var snapshots [][]Stage
			flasher := &progressFlasher{fakeDeviceFlasher: fakeDeviceFlasher{chip: "ESP32-C3"}}
			flasher.check = func(want string) {
				if !strings.Contains(latest, want) {
					t.Fatalf("work started before detail %q: %q", want, latest)
				}
			}
			if failWrite {
				flasher.flashError = errors.New("write failed")
			}
			session := &fakeREPL{files: map[string][]byte{"/config/node_config.json": []byte(`{"version":"older"}`)}}
			manager := ReleaseManager{Assets: files, BackupDir: t.TempDir(),
				Discover: func(context.Context) ([]Device, error) { return nil, nil },
				OpenFlasher: func(string, InstallConfig) (deviceFlasher, error) {
					flasher.check("Connect to ESP bootloader")
					return flasher, nil
				},
				OpenREPL: func(context.Context, string, REPLConfig) (replSession, error) { return session, nil },
			}
			run := manager.Update
			if install {
				run = manager.Install
			}
			_, err := run(context.Background(), updateTestTarget(), func(stages []Stage) {
				snapshots = append(snapshots, stages)
				for _, stage := range stages {
					if stage.Name == "Install and verify release firmware" && stage.State == StageRunning {
						if stage.CanReconnect {
							t.Fatal("flashing must not permit unplugging")
						}
						latest = stage.Detail
					}
				}
			})
			if !errors.Is(err, flasher.flashError) {
				t.Fatalf("unexpected result: %v", err)
			}
			// Saved events must retain intermediate progress after completion.
			found := false
			for _, stages := range snapshots {
				for _, stage := range stages {
					found = found || strings.Contains(stage.Detail, "50% transferred")
				}
			}
			if !found {
				t.Fatal("intermediate firmware detail was lost")
			}
		}
	}
}
