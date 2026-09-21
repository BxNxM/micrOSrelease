package selfupdate

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/iotest"
)

func TestExecutableInstallKeepsBackup(t *testing.T) {
	binary := []byte("new microsctl payload")
	var err error
	target := filepath.Join(t.TempDir(), "microsctl")
	if err = os.WriteFile(target, []byte("previous executable"), 0755); err != nil {
		t.Fatal(err)
	}
	backup, err := (ExecutableInstaller{Path: target}).Install(context.Background(), bytes.NewReader(binary))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(backup) != ".microsctl-backup" {
		t.Fatalf("unexpected backup path: %s", backup)
	}
	old, err := os.ReadFile(backup)
	if err != nil || string(old) != "previous executable" {
		t.Fatal("backup not retained", err)
	}
	installed, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(installed, binary) {
		t.Fatal("replacement differs", err)
	}
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(target)
		if info.Mode().Perm()&0111 == 0 {
			t.Fatal("lost executable permission")
		}
	}
	files, _ := filepath.Glob(filepath.Join(filepath.Dir(target), ".microsctl-update-*"))
	if len(files) != 0 {
		t.Fatal("staging file leaked")
	}
}

func TestRepeatedUpdatesKeepOnlyPreviousExecutable(t *testing.T) {
	directory, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(directory, "microsctl")
	if err := os.WriteFile(target, []byte("version 1"), 0755); err != nil {
		t.Fatal(err)
	}
	for version := 2; version <= 4; version++ {
		if runtime.GOOS == "windows" {
			// Simulate an image left behind by an exited Windows process.
			if err := os.WriteFile(filepath.Join(directory, ".microsctl-backup.running.exe"), []byte("old image"), 0755); err != nil {
				t.Fatal(err)
			}
		}
		payload := fmt.Sprintf("version %d", version)
		backup, err := (ExecutableInstaller{Path: target}).Install(context.Background(), strings.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		if backup != filepath.Join(directory, ".microsctl-backup") {
			t.Fatalf("unexpected backup path: %s", backup)
		}
		previous, err := os.ReadFile(backup)
		if err != nil || string(previous) != fmt.Sprintf("version %d", version-1) {
			t.Fatalf("backup=%q error=%v", previous, err)
		}
		installed, err := os.ReadFile(target)
		if err != nil || string(installed) != payload {
			t.Fatalf("installed=%q error=%v", installed, err)
		}
		files, err := os.ReadDir(directory)
		if err != nil || len(files) != 2 {
			t.Fatalf("updates accumulated files: %v error=%v", files, err)
		}
	}
}

func TestCancelledBackupPreservesPreviousCopy(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "microsctl")
	backup := filepath.Join(directory, ".microsctl-backup")
	for path, content := range map[string]string{target: "current", backup: "previous"} {
		if err := os.WriteFile(path, []byte(content), 0755); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := backupExecutable(ctx, target, 0755); err != context.Canceled {
		t.Fatalf("expected cancellation, got %v", err)
	}
	previous, err := os.ReadFile(backup)
	if err != nil || string(previous) != "previous" {
		t.Fatalf("previous backup changed: %q error=%v", previous, err)
	}
	files, err := os.ReadDir(directory)
	if err != nil || len(files) != 2 {
		t.Fatalf("temporary backup leaked: %v error=%v", files, err)
	}
}

func TestRejectedDownloadsLeaveExecutableUntouched(t *testing.T) {
	for _, kind := range []string{"empty", "error", "cancel"} {
		t.Run(kind, func(t *testing.T) {
			target := filepath.Join(t.TempDir(), "microsctl")
			os.WriteFile(target, []byte("old"), 0755)
			backup := filepath.Join(filepath.Dir(target), ".microsctl-backup")
			if err := os.WriteFile(backup, []byte("previous"), 0755); err != nil {
				t.Fatal(err)
			}
			var source io.Reader = strings.NewReader("download")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch kind {
			case "empty":
				source = strings.NewReader("")
			case "error":
				source = io.MultiReader(source, iotest.ErrReader(io.ErrUnexpectedEOF))
			case "cancel":
				cancel()
			}
			if _, err := (ExecutableInstaller{Path: target}).Install(ctx, source); err == nil {
				t.Fatal("accepted invalid download")
			}
			kept, _ := os.ReadFile(target)
			if string(kept) != "old" {
				t.Fatal("current binary changed")
			}
			files, _ := os.ReadDir(filepath.Dir(target))
			if len(files) != 2 {
				t.Fatal("temporary files leaked")
			}
			previous, err := os.ReadFile(backup)
			if err != nil || string(previous) != "previous" {
				t.Fatalf("previous backup changed: %q error=%v", previous, err)
			}
		})
	}
}

func TestRestartPreservesArgumentsDirectoryAndEnvironment(t *testing.T) {
	if os.Getenv("MICROSCTL_RESTART_FIXTURE") == "1" {
		if os.Args[len(os.Args)-1] == "restarted" {
			cwd, _ := os.Getwd()
			if cwd != os.Getenv("MICROSCTL_RESTART_DIR") {
				os.Exit(2)
			}
			fmt.Print("restart-ok:" + os.Getenv("MICROSCTL_RESTART_VALUE"))
			os.Exit(0)
		}
		executable, _ := os.Executable()
		binary, err := os.ReadFile(executable)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if _, err = (ExecutableInstaller{Path: executable}).Install(context.Background(), bytes.NewReader(binary)); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if err := Restart(executable, []string{"-test.run=^TestRestartPreservesArgumentsDirectoryAndEnvironment$", "--", "restarted"}); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return
	}
	executable, _ := os.Executable()
	directory, _ := filepath.EvalSymlinks(t.TempDir())
	binary, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(directory, "microsctl-fixture.exe")
	if err = os.WriteFile(fixture, binary, 0755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(fixture, "-test.run=^TestRestartPreservesArgumentsDirectoryAndEnvironment$")
	command.Dir = directory
	command.Env = append(os.Environ(), "MICROSCTL_RESTART_FIXTURE=1", "MICROSCTL_RESTART_VALUE=retained", "MICROSCTL_RESTART_DIR="+command.Dir)
	output, err := command.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "restart-ok:retained") {
		t.Fatalf("restart failed: %s %v", output, err)
	}
}
