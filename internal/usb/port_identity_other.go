//go:build !darwin && !linux && !windows

package usb

import "context"

func detailedUSBPorts(context.Context) ([]Device, error) { return nil, nil }
