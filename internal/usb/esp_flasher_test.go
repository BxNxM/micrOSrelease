package usb

import (
	"testing"

	"tinygo.org/x/espflasher/pkg/espflasher"
)

func TestInstallFlasherSelectsResetForTransport(t *testing.T) {
	for _, tc := range []struct {
		name, port, chip, reset string
		want                    espflasher.ResetMode
	}{
		{"ESP32 CP2102", "/dev/cu.SLAB_USBtoUART", "esp32", "auto", espflasher.ResetDefault},
		{"ESP32 Windows", "COM7", "esp32", "auto", espflasher.ResetDefault},
		{"ESP8266 Windows", "COM7", "esp8266", "auto", espflasher.ResetDefault},
		{"C6 native USB", "/dev/cu.usbmodem2101", "esp32c6", "auto", espflasher.ResetAuto},
		{"C6 Windows USB", "COM7", "esp32c6", "auto", espflasher.ResetAuto},
		{"C6 UART bridge", "/dev/ttyUSB0", "esp32c6", "auto", espflasher.ResetDefault},
		{"explicit no reset", "/dev/cu.SLAB_USBtoUART", "esp32", "no-reset", espflasher.ResetNoReset},
		{"explicit JTAG", "/dev/ttyUSB0", "esp32c6", "usb-jtag", espflasher.ResetUSBJTAG},
		{"explicit UART", "/dev/cu.usbmodem1", "esp32c6", "default", espflasher.ResetDefault},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts, err := flasherOptions(Device{Port: tc.port}, InstallConfig{Chip: tc.chip, ResetMode: tc.reset, InitialBaud: 115200, FlashBaud: 460800, Compress: true})
			if err != nil {
				t.Fatal(err)
			}
			if opts.ResetMode != tc.want || opts.SkipStub || opts.BaudRate != 115200 || opts.FlashBaudRate != 460800 || !opts.Compress {
				t.Fatalf("incorrect flashing options: %+v", opts)
			}
		})
	}
}

func TestFlashAndProbeUseSameUSBTransport(t *testing.T) {
	for _, chip := range []string{"esp32c3", "esp32c6", "esp32s3"} {
		for _, vid := range []string{"10C4", "1A86", "0403", "067B", "303A"} {
			device := Device{Port: "COM7", USBVID: vid}
			opts, err := flasherOptions(device, InstallConfig{Chip: chip, ResetMode: "auto"})
			if err != nil {
				t.Fatal(err)
			}
			if want := probeOptions(device).ResetMode; opts.ResetMode != want {
				t.Errorf("%s on USB vendor %s: flash reset %v differs from probe %v", chip, vid, opts.ResetMode, want)
			}
		}
	}
}
