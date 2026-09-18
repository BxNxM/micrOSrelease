package usb

import (
	"errors"
	"io"
	"os"
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
