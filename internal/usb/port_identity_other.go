//go:build !darwin && !linux

package usb

import "context"

func detailedUSBPorts(context.Context) ([]Device, error) { return nil, nil }
