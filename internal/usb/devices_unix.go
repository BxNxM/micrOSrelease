//go:build !windows

package usb

import (
	"context"
	"os"
	"path/filepath"
)

func discoverDevices(ctx context.Context) ([]Device, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir("/dev")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		names = append(names, entry.Name())
	}
	return enrichUSBIdentity(ctx, matchingUnixDevices(names, "/dev")), nil
}

func joinDevicePath(directory, name string) string {
	return filepath.Join(directory, name)
}
