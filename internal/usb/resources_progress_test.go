package usb

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

type progressREPL struct {
	fakeREPL
	beforeWrite func(string)
}

func (s *progressREPL) WriteFileAtomic(ctx context.Context, name string, data []byte) error {
	s.beforeWrite(name)
	return s.fakeREPL.WriteFileAtomic(ctx, name, data)
}

func TestResourceProgressDuringInstallAndUpdate(t *testing.T) {
	for _, operation := range []string{"install", "update", "refresh"} {
		t.Run(operation, func(t *testing.T) {
			files := installTestFS()
			files["modules/main.py"] = &fstest.MapFile{Data: []byte("startup")}
			files["modules/modules/feature.mpy"] = &fstest.MapFile{Data: []byte("module")}
			version := "older"
			if operation == "refresh" {
				version = "3.6.0-0"
			}
			var snapshots [][]Stage
			var details []string
			session := &progressREPL{fakeREPL: fakeREPL{files: map[string][]byte{
				"/config/node_config.json": []byte(`{"version":"` + version + `"}`),
			}}}
			session.beforeWrite = func(name string) {
				if name == "/config/node_config.json" {
					return
				}
				// The user must see the current file while WriteFileAtomic is
				// still running, not only once the transfer has finished.
				if len(details) == 0 || !strings.HasSuffix(details[len(details)-1], name) {
					t.Fatalf("upload %s started before its progress event: %v", name, details)
				}
			}
			manager := ReleaseManager{
				Assets: files, BackupDir: t.TempDir(),
				Discover:    func(context.Context) ([]Device, error) { return nil, nil },
				OpenREPL:    func(context.Context, string, REPLConfig) (replSession, error) { return session, nil },
				OpenFlasher: func(string, InstallConfig) (deviceFlasher, error) { return &fakeDeviceFlasher{chip: "ESP32-C3"}, nil },
			}
			run := manager.Update
			if operation == "install" {
				run = manager.Install
			}
			_, err := run(context.Background(), updateTestTarget(), func(stages []Stage) {
				snapshots = append(snapshots, stages)
				for _, stage := range stages {
					if stage.State == StageRunning && strings.HasPrefix(stage.Detail, "Uploading ") {
						if stage.CanReconnect {
							t.Fatal("upload must never permit unplugging")
						}
						details = append(details, stage.Detail)
					}
				}
			})
			if err != nil {
				t.Fatal(err)
			}
			want := []string{"Uploading 1/2 · /modules/feature.mpy", "Uploading 2/2 · /main.py"}
			if !reflect.DeepEqual(details, want) {
				t.Fatalf("progress = %v, want %v", details, want)
			}
			// Emitted snapshots must not be mutated by later stage updates.
			var saved []string
			for _, stages := range snapshots {
				for _, stage := range stages {
					if stage.State == StageRunning && strings.HasPrefix(stage.Detail, "Uploading ") {
						saved = append(saved, stage.Detail)
					}
				}
			}
			if !reflect.DeepEqual(saved, want) {
				t.Fatalf("progress snapshots changed: %v", saved)
			}
		})
	}
}

func TestResourceProgressStopsOnFailureOrCancellation(t *testing.T) {
	resources := []preparedResource{{target: "/first.py"}, {target: "/main.py"}}
	for _, cancelFirst := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		session := &fakeREPL{writeError: errors.New("verification failed")}
		want := session.writeError
		if cancelFirst {
			cancel()
			want = context.Canceled
		}
		var details []string
		err := copyResources(ctx, session, resources, func(detail string) { details = append(details, detail) })
		cancel()
		if !errors.Is(err, want) {
			t.Fatalf("error = %v, want %v", err, want)
		}
		if cancelFirst && len(details) != 0 || !cancelFirst && !reflect.DeepEqual(details, []string{"Uploading 1/2 · /first.py"}) {
			t.Fatalf("unexpected progress after failure/cancellation: %v", details)
		}
	}
}
