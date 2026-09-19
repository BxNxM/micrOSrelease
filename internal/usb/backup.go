package usb

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/micros/microsctl/internal/micropython"
)

type savedFile struct {
	micropython.FileInfo
	data []byte
}

// snapshotDevice must finish before erase. Never silently skip unreadable,
// oversized, hidden, or custom files: an incomplete backup is not safe to flash.
func snapshotDevice(ctx context.Context, session replSession) ([]savedFile, error) {
	entries, err := session.ListFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("inventory device backup: %w", err)
	}
	var snapshot []savedFile
	var total int64
	seen := make(map[string]bool)
	for _, entry := range entries {
		if !validDevicePath(entry.Path) || seen[entry.Path] || entry.Size < 0 {
			return nil, fmt.Errorf("invalid backup entry %q", entry.Path)
		}
		seen[entry.Path] = true
		file := savedFile{FileInfo: entry}
		if !entry.Directory {
			total += entry.Size
			if entry.Size > micropython.MaxFileSize || total > 64<<20 {
				return nil, fmt.Errorf("backup limit exceeded at %s (1 MiB per file, 64 MiB total); flash was not erased", entry.Path)
			}
			file.data, err = session.ReadFile(ctx, entry.Path)
			if err != nil {
				return nil, fmt.Errorf("back up %s: %w", entry.Path, err)
			}
			if int64(len(file.data)) != entry.Size {
				return nil, fmt.Errorf("file changed during backup: %s", entry.Path)
			}
		}
		snapshot = append(snapshot, file)
	}
	sort.Slice(snapshot, func(i, j int) bool { return snapshot[i].Path < snapshot[j].Path })
	return snapshot, nil
}

// archiveDeviceState produces a durable, private ZIP before destructive work.
// Filenames inside the archive are relative; no device path is used on the host.
func archiveDeviceState(directory, identity string, snapshot []savedFile) (string, error) {
	if directory == "" {
		return "", fmt.Errorf("a backup directory is required before replacing firmware")
	}
	return archiveBackup(directory, identity, "filesystem.zip", func(file io.Writer) error {
		writer := zip.NewWriter(file)
		for _, entry := range snapshot {
			name := strings.TrimPrefix(entry.Path, "/")
			if entry.Directory {
				name += "/"
			}
			header := &zip.FileHeader{Name: name, Method: zip.Deflate}
			header.SetMode(0o600)
			if entry.Directory {
				header.SetMode(os.ModeDir | 0o700)
			}
			out, err := writer.CreateHeader(header)
			if err != nil {
				return err
			}
			if _, err := out.Write(entry.data); err != nil {
				return err
			}
		}
		return writer.Close()
	})
}

func restoreDeviceState(ctx context.Context, session replSession, snapshot []savedFile, resources []preparedResource, config REPLConfig) error {
	// Release resources replace their own destinations. Preserve all other files,
	// including offloaded config, custom modules, pin maps, and user data.
	replaced := make(map[string]bool)
	for _, resource := range resources {
		replaced[resource.target] = true
	}
	for _, name := range config.ConfigPaths {
		replaced[name] = true
	}
	replaced[config.RestorePath] = true
	for _, entry := range snapshot {
		if entry.Directory {
			if err := session.MkdirAll(ctx, entry.Path); err != nil {
				return err
			}
		} else if !replaced[entry.Path] {
			if err := session.WriteFileAtomic(ctx, entry.Path, entry.data); err != nil {
				return fmt.Errorf("restore %s: %w", entry.Path, err)
			}
		}
	}
	return nil
}

func archiveNodeConfig(directory string, parsed map[string]any, data []byte) (string, error) {
	if directory == "" {
		return "", nil
	}
	identity, _ := parsed["devfid"].(string)
	return archiveBackup(directory, identity, "node_config.json", func(file io.Writer) error {
		_, err := file.Write(data)
		return err
	})
}

// archiveBackup publishes a private file only after the complete payload is
// written, synced and closed. Both backup formats share the same cleanup rules.
func archiveBackup(directory, identity, suffix string, write func(io.Writer) error) (string, error) {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", fmt.Errorf("create backup directory: %w", err)
	}
	file, err := os.CreateTemp(directory, ".micros-backup-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create %s backup: %w", suffix, err)
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if err := write(file); err != nil {
		return "", fmt.Errorf("write %s backup: %w", suffix, err)
	}
	if err := file.Sync(); err != nil {
		return "", fmt.Errorf("sync %s backup: %w", suffix, err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("close %s backup: %w", suffix, err)
	}
	identity = safeFilename(identity)
	if identity == "" {
		identity = "device"
	}
	name := filepath.Join(directory, fmt.Sprintf("%s-%s-%s", identity, time.Now().UTC().Format("20060102T150405.000000000Z"), suffix))
	if err := publishBackup(file.Name(), name); err != nil {
		return "", fmt.Errorf("archive %s: %w", suffix, err)
	}
	return name, nil
}

func safeFilename(value string) string {
	return strings.Map(func(character rune) rune {
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || character == '-' || character == '_' {
			return character
		}
		return '-'
	}, strings.TrimSpace(value))
}
