package usb

import (
	"regexp"
	"sort"
	"strings"
)

var unixDeviceIdentifiers = []string{
	"wchusbserial",
	"SLAB_USBtoUART",
	"USB0",
	"usbserial",
	"usbmodem",
	"ttyACM",
	"ttyUSB",
}

// PnP IDs cover native Espressif USB even with localized/generic CDC names.
var windowsDeviceIdentifiers = []string{"CP210", "CH340", "CH343", "CH9102", "USB JTAG/serial", `VID_303A&`}

var windowsCOMPort = regexp.MustCompile(`(?i)\((COM[0-9]+)\)`)

func matchingUnixDevices(names []string, deviceDirectory string) []Device {
	// macOS publishes tty.* and cu.* for the same endpoint. Prefer the
	// callout interface when present; Linux continues to use ttyUSB/ttyACM.
	callouts := make(map[string]bool)
	for _, name := range names {
		if strings.HasPrefix(name, "cu.") {
			callouts[strings.TrimPrefix(name, "cu.")] = true
		}
	}
	var devices []Device
	for _, name := range names {
		if (!strings.HasPrefix(name, "tty") && !strings.HasPrefix(name, "cu.")) || !containsIdentifier(name, unixDeviceIdentifiers) {
			continue
		}
		if strings.HasPrefix(name, "tty.") && callouts[strings.TrimPrefix(name, "tty.")] {
			continue
		}
		devices = append(devices, Device{Port: joinDevicePath(deviceDirectory, name)})
	}
	sort.Slice(devices, func(i, j int) bool { return devices[i].Port < devices[j].Port })
	return devices
}

func matchingWindowsDevices(descriptions []string) []Device {
	seen := make(map[string]bool)
	var devices []Device
	for _, description := range descriptions {
		if !containsIdentifier(description, windowsDeviceIdentifiers) {
			continue
		}
		match := windowsCOMPort.FindStringSubmatch(description)
		if len(match) != 2 {
			continue
		}
		port := strings.ToUpper(match[1])
		if !seen[port] {
			seen[port] = true
			devices = append(devices, Device{Port: port})
		}
	}
	sort.Slice(devices, func(i, j int) bool { return devices[i].Port < devices[j].Port })
	return devices
}

func containsIdentifier(value string, identifiers []string) bool {
	value = strings.ToLower(value)
	for _, identifier := range identifiers {
		if strings.Contains(value, strings.ToLower(identifier)) {
			return true
		}
	}
	return false
}
