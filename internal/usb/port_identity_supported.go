//go:build linux || windows

package usb

import (
	"context"
	"go.bug.st/serial/enumerator"
)

func detailedUSBPorts(ctx context.Context) ([]Device, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ports, err := enumerator.GetDetailedPortsList() // Passive metadata only; never open/reset devices.
	if err != nil {
		return nil, err
	}
	var devices []Device
	for _, p := range ports {
		devices = append(devices, Device{Port: p.Name, USBSerial: p.SerialNumber, USBVID: p.VID, USBPID: p.PID})
	}
	return devices, nil
}
