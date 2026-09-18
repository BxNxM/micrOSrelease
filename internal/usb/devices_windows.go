//go:build windows

package usb

import (
	"context"
	"os/exec"
	"strings"
)

const windowsSerialPortsCommand = `Get-CimInstance Win32_PnPEntity | Where-Object { $_.Name -match '\(COM[0-9]+\)' } | ForEach-Object { $_.Name + ' ' + $_.PNPDeviceID }`

func discoverDevices(ctx context.Context) ([]Device, error) {
	output, err := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", windowsSerialPortsCommand).Output()
	if err != nil {
		return nil, err
	}
	return enrichUSBIdentity(ctx, matchingWindowsDevices(strings.Split(string(output), "\n"))), nil
}

func joinDevicePath(directory, name string) string {
	return directory + `\` + name
}
