package usb

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/micros/microsctl/internal/micropython"
)

func TestUpdateStopsWithoutErasingWhenRuntimeOrBootloaderIsUnavailable(t *testing.T) {
	for _, chip := range []string{"esp32", "esp32s3", "esp32c6"} {
		for _, bootloaderFailure := range []bool{false, true} {
			failure := "runtime unavailable"
			if bootloaderFailure {
				failure = "bootloader unavailable"
			}
			t.Run(chip+"/"+failure, func(t *testing.T) {
				files := installTestFS()
				manifest := files["frameworks/esp32c3/install.json"]
				manifest.Data = []byte(strings.ReplaceAll(string(manifest.Data), "esp32c3", chip))
				originalConfig := `{"version":"older","devfid":"keep-config"}`
				session := &fakeREPL{
					files:   map[string][]byte{"/config/node_config.json": []byte(originalConfig), "/data/custom.txt": []byte("keep-file")},
					runtime: micropython.RuntimeInfo{Machine: chip, MicroPython: "1.28.0"},
				}
				cause := errors.New("USB port disappeared during mode transition")
				flasherOpens := 0
				manager := ReleaseManager{Assets: files, BackupDir: t.TempDir(),
					Discover: func(context.Context) ([]Device, error) { return nil, nil },
					OpenREPL: func(context.Context, string, REPLConfig) (replSession, error) {
						if !bootloaderFailure {
							return nil, cause
						}
						return session, nil
					},
					OpenFlasher: func(string, InstallConfig) (deviceFlasher, error) {
						flasherOpens++
						return nil, cause
					},
				}
				_, err := manager.Update(context.Background(), updateTestTarget())
				if !errors.Is(err, cause) || !strings.Contains(err.Error(), "flash was not erased") || len(session.writes) != 0 {
					t.Fatalf("unsafe transition failure: %v, writes=%v", err, session.writes)
				}
				if !bootloaderFailure {
					if flasherOpens != 0 || !strings.Contains(err.Error(), "Reboot normally") {
						t.Fatal("missing runtime must stop before flashing and explain normal boot")
					}
					return
				}
				if flasherOpens != 1 || !session.closed {
					t.Fatal("failed transition must release REPL and never retry flashing automatically")
				}
				archives, globErr := filepath.Glob(filepath.Join(manager.BackupDir, "*-filesystem.zip"))
				if globErr != nil || len(archives) != 1 || !strings.Contains(err.Error(), archives[0]) {
					t.Fatalf("failure omitted persistent backup: %v, %v", archives, err)
				}
				if info, statErr := os.Stat(archives[0]); statErr != nil || info.Size() == 0 {
					t.Fatal("persistent backup is missing or empty")
				}
				if string(session.files["/config/node_config.json"]) != originalConfig || string(session.files["/data/custom.txt"]) != "keep-file" {
					t.Fatal("transition failure changed original files")
				}
			})
		}
	}
}

type failingFinalReset struct {
	fakeREPL
	cause  error
	resets int
}

func (s *failingFinalReset) Reset() error {
	s.resets++
	return s.cause
}

func TestFinalResetFailureKeepsVerifiedFilesAndExplainsRecovery(t *testing.T) {
	for _, install := range []bool{false, true} {
		files := installTestFS()
		files["modules/main.py"] = &fstest.MapFile{Data: []byte("verified startup")}
		cause := errors.New("reset command could not be sent")
		session := &failingFinalReset{cause: cause, fakeREPL: fakeREPL{
			files: map[string][]byte{"/config/node_config.json": []byte(`{"version":"older"}`)},
		}}
		flashes := 0
		manager := ReleaseManager{Assets: files, BackupDir: t.TempDir(),
			Discover: func(context.Context) ([]Device, error) { return nil, nil },
			OpenREPL: func(context.Context, string, REPLConfig) (replSession, error) { return session, nil },
			OpenFlasher: func(string, InstallConfig) (deviceFlasher, error) {
				flashes++
				return &fakeDeviceFlasher{chip: "ESP32-C3"}, nil
			},
		}
		run := manager.Update
		if install {
			run = manager.Install
		}
		_, err := run(context.Background(), updateTestTarget())
		if !errors.Is(err, cause) || !strings.Contains(err.Error(), "files are verified; reboot normally without holding BOOT") {
			t.Fatalf("missing reboot recovery instructions: %v", err)
		}
		if flashes != 1 || session.resets != 1 || string(session.writes["/main.py"]) != "verified startup" || !session.closed {
			t.Fatal("reset failure must preserve the completed transfer and stop without retrying")
		}
		if !install && !strings.Contains(err.Error(), "device backup:") {
			t.Fatal("update reset failure lost the backup path")
		}
	}
}
