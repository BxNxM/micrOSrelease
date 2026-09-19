package usb

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestFailedBackupDoesNotPublishPartialArchive(t *testing.T) {
	directory := t.TempDir()
	want := errors.New("backup interrupted")
	name, err := archiveBackup(directory, "device", "filesystem.zip", func(file io.Writer) error {
		if _, err := file.Write([]byte("incomplete archive")); err != nil {
			t.Fatal(err)
		}
		return want
	})
	if name != "" || !errors.Is(err, want) {
		t.Fatalf("failed backup reported as usable: %q, %v", name, err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 0 {
		t.Fatalf("partial backup left on disk: %v, %v", entries, err)
	}
}

func TestBackupPublishesInNewDirectoryHierarchy(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "new", "nested", "backups")
	name, err := archiveBackup(directory, "device", "node_config.json", func(file io.Writer) error {
		_, err := io.WriteString(file, "complete backup")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(name)
	if err != nil || string(data) != "complete backup" {
		t.Fatalf("published backup unavailable: %q, %v", data, err)
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 || entries[0].Name() != filepath.Base(name) {
		t.Fatalf("unexpected backup publication: %v, %v", entries, err)
	}
}
