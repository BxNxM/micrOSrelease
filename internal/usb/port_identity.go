package usb

import (
	"context"
	"strings"
)

// macOS exposes two names for the same endpoint. Discovery prefers cu.*.
func samePort(a, b string) bool {
	canonical := func(s string) string { return strings.Replace(s, "/dev/tty.", "/dev/cu.", 1) }
	return canonical(a) == canonical(b)
}

func enrichUSBIdentity(ctx context.Context, devices []Device) []Device {
	details, err := detailedUSBPorts(ctx)
	if err != nil {
		return devices
	} // Identity is optional; original-port reconnect still works.
	for i := range devices {
		for _, d := range details {
			if samePort(devices[i].Port, d.Port) {
				devices[i].USBSerial, devices[i].USBVID, devices[i].USBPID = d.USBSerial, d.USBVID, d.USBPID
				devices[i].USBLocation = d.USBLocation
				break
			}
		}
	}
	return devices
}

func reconnectCandidates(target Device, devices []Device) []Device {
	var matches []Device
	for _, device := range devices {
		// Firmware can change the port, product ID, and USB serial string.
		// When available, bind to the physical connection instead of guessing
		// which newly appeared endpoint is the board being restored.
		if target.USBLocation != "" {
			if device.USBLocation == target.USBLocation && strings.EqualFold(device.USBVID, target.USBVID) {
				matches = append(matches, device)
			}
		} else if target.USBSerial != "" {
			if device.USBSerial == target.USBSerial && strings.EqualFold(device.USBVID, target.USBVID) {
				matches = append(matches, device)
			}
		} else if samePort(device.Port, target.Port) {
			matches = append(matches, device)
		}
	}
	// Some drivers expose multiple endpoints for one USB device. Retain the
	// selected endpoint when present; otherwise never guess among matches.
	if len(matches) > 1 {
		for _, device := range matches {
			if samePort(device.Port, target.Port) {
				return []Device{device}
			}
		}
		return nil
	}
	return matches
}
