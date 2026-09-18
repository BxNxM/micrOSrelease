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

func TestMatchingWindowsDevices(t *testing.T) {
	descriptions := []string{
		"Standard Serial over Bluetooth link (COM2)\r",
		"Silicon Labs CP210x USB to UART Bridge (COM7)\r",
		"USB-SERIAL CH340 (COM12)\r",
		"USB-SERIAL CH343 (COM3)\r",
		"USB Enhanced-SERIAL CH9102 (com9)\r",
		"USB-SERIAL CH340 (COM12)\r",
		"USB JTAG/serial debug unit (COM8) USB\\VID_303A&PID_1001",
		"USB Serial Device (COM10) USB\\VID_303A&PID_1001",
		"Dispositivo serie USB (COM11) USB\\VID_303A&PID_1001",
		"USB Serial Device (COM13) USB\\VID_1234&PID_5678",
	}
	want := []Device{{Port: "COM10"}, {Port: "COM11"}, {Port: "COM12"}, {Port: "COM3"}, {Port: "COM7"}, {Port: "COM8"}, {Port: "COM9"}}
	if got := matchingWindowsDevices(descriptions); !reflect.DeepEqual(got, want) {
		t.Fatalf("matchingWindowsDevices() = %#v, want %#v", got, want)
	}
}
