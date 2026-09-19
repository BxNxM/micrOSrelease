//go:build !windows

package usb

import (
	"fmt"
	"os"
	"path/filepath"
)

// Sync the renamed entry and its ancestors, including any directories just
// created by MkdirAll. A file sync alone does not make its name durable.
func publishBackup(temporary, name string) error {
	if err := os.Rename(temporary, name); err != nil {
		return err
	}
	directory, err := filepath.Abs(filepath.Dir(name))
	if err != nil {
		return err
	}
	for {
		file, err := os.Open(directory)
		if err != nil {
			return err
		}
		syncErr := file.Sync()
		closeErr := file.Close()
		if syncErr != nil {
			return fmt.Errorf("sync backup directory %s: %w", directory, syncErr)
		}
		if closeErr != nil {
			return closeErr
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return nil
		}
		directory = parent
	}
}
