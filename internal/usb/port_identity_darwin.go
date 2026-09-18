package usb

import (
	"context"
	"encoding/xml"
	"fmt"
	"os/exec"
	"strconv"
	"time"
)

// ioreg is part of macOS. Reading its passive registry also works in the
// CGO_ENABLED=0 release builds, without opening any USB device.
func detailedUSBPorts(ctx context.Context) ([]Device, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	data, err := exec.CommandContext(ctx, "/usr/sbin/ioreg", "-a", "-l", "-r", "-c", "IOUSBHostDevice").Output()
	if err != nil {
		return nil, err
	}
	return parseUSBRegistry(data)
}

type registryValue struct {
	XMLName xml.Name
	Text    string          `xml:",chardata"`
	Values  []registryValue `xml:",any"`
}

func (v registryValue) property(name string) registryValue {
	for i := 0; i+1 < len(v.Values); i += 2 {
		if v.Values[i].XMLName.Local == "key" && v.Values[i].Text == name {
			return v.Values[i+1]
		}
	}
	return registryValue{}
}

func parseUSBRegistry(data []byte) ([]Device, error) {
	var root registryValue
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	var devices []Device
	var walk func(registryValue, Device)
	walk = func(node registryValue, identity Device) {
		class := node.property("IOObjectClass").Text
		if class == "IOUSBHostDevice" || class == "IOUSBDevice" {
			identity = Device{USBSerial: node.property("kUSBSerialNumberString").Text}
			if identity.USBSerial == "" {
				identity.USBSerial = node.property("USB Serial Number").Text
			}
			vid, _ := strconv.ParseUint(node.property("idVendor").Text, 0, 16)
			pid, _ := strconv.ParseUint(node.property("idProduct").Text, 0, 16)
			identity.USBVID, identity.USBPID = fmt.Sprintf("%04X", vid), fmt.Sprintf("%04X", pid)
		}
		if port := node.property("IOCalloutDevice").Text; port != "" {
			identity.Port = port
			devices = append(devices, identity)
		}
		for _, child := range node.property("IORegistryEntryChildren").Values {
			walk(child, identity)
		}
	}
	for _, array := range root.Values {
		if array.XMLName.Local == "array" {
			for _, node := range array.Values {
				walk(node, Device{})
			}
		}
	}
	return devices, nil
}
