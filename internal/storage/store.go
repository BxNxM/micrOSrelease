// Package storage owns persistent application data outside the executable.
package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/micros/micros-release/internal/network"
)

type Store struct{ Root string }

// Open initializes a platform-specific application directory, or an explicit
// directory for portable installations. Firmware files live alongside the cache.
func Open(root string) (*Store, error) {
	if root == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			return nil, err
		}
		root = filepath.Join(base, "micros-release")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	s := &Store{Root: root}
	if err := os.MkdirAll(s.FirmwareDir(), 0700); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) FirmwareDir() string { return filepath.Join(s.Root, "firmware") }

func (s *Store) LoadDevices() ([]network.Device, error) {
	data, err := os.ReadFile(filepath.Join(s.Root, "devices.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var cache struct {
		Version int
		Devices []network.Device
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("read device cache: %w", err)
	}
	if cache.Version != 1 {
		return nil, fmt.Errorf("unsupported device cache version %d", cache.Version)
	}
	return cache.Devices, nil
}

// SaveDevices replaces the cache only after the complete JSON has been written.
func (s *Store) SaveDevices(devices []network.Device) error {
	data, err := json.MarshalIndent(struct {
		Version int
		Devices []network.Device
	}{1, devices}, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(s.Root, ".devices-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), filepath.Join(s.Root, "devices.json"))
}
