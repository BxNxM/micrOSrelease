package usb

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestMatchingUnixDevices(t *testing.T) {
	names := []string{
		"tty.Bluetooth-Incoming-Port",
		"cu.usbserial-standalone",
		"cu.usbmodem101",
		"cu.Bluetooth-Incoming-Port",
		"tty.wchusbserial1410",
		"tty.SLAB_USBtoUART",
		"ttyACM0",
		"ttyUSB0",
		"tty.usbmodem101",
	}
	want := []Device{
		{Port: filepath.Join("/devices", "cu.usbmodem101")},
		{Port: filepath.Join("/devices", "cu.usbserial-standalone")},
		{Port: filepath.Join("/devices", "tty.SLAB_USBtoUART")},
		{Port: filepath.Join("/devices", "tty.wchusbserial1410")},
		{Port: filepath.Join("/devices", "ttyACM0")},
		{Port: filepath.Join("/devices", "ttyUSB0")},
	}
	if got := matchingUnixDevices(names, "/devices"); !reflect.DeepEqual(got, want) {
		t.Fatalf("matchingUnixDevices() = %#v, want %#v", got, want)
	}
}
