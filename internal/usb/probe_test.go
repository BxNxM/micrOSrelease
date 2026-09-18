package usb

import (
	"context"
	"errors"
	"net"
	"reflect"
	"testing"

	"tinygo.org/x/espflasher/pkg/espflasher"
)

func TestProbeResetStrategyMatchesUSBTransport(t *testing.T) {
	for _, tc := range []struct {
		name   string
		device Device
		reset  espflasher.ResetMode
	}{
		{"CP210x metadata", Device{Port: "COM7", USBVID: "10c4"}, espflasher.ResetDefault},
		{"WCH metadata", Device{Port: "COM8", USBVID: "1a86"}, espflasher.ResetDefault},
		{"FTDI metadata", Device{Port: "COM9", USBVID: "0403"}, espflasher.ResetDefault},
		{"Prolific metadata", Device{Port: "COM10", USBVID: "067B"}, espflasher.ResetDefault},
		{"Silabs macOS", Device{Port: "/dev/cu.SLAB_USBtoUART"}, espflasher.ResetDefault},
		{"Apple macOS", Device{Port: "/dev/cu.usbserial-0001"}, espflasher.ResetDefault},
		{"WCH macOS", Device{Port: "/dev/cu.wchusbserial1"}, espflasher.ResetDefault},
		{"Linux UART", Device{Port: "/dev/ttyUSB0"}, espflasher.ResetDefault},
		{"native C6", Device{Port: "/dev/cu.usbmodem2101", USBVID: "303a"}, espflasher.ResetAuto},
		{"native USB overrides name", Device{Port: "/dev/ttyUSB0", USBVID: "303A"}, espflasher.ResetAuto},
		{"unknown modem", Device{Port: "/dev/cu.usbmodem2101"}, espflasher.ResetAuto},
		{"unknown Windows", Device{Port: "COM7"}, espflasher.ResetAuto},
	} {
		t.Run(tc.name, func(t *testing.T) {
			options := probeOptions(tc.device)
			if options.ResetMode != tc.reset || !options.SkipStub {
				t.Fatalf("probe options: %+v; want reset %v and ROM-only discovery", options, tc.reset)
			}
		})
	}
}

type fakeDeviceProbe struct {
	chip     string
	flashID  uint16
	flashErr error
	mac      net.HardwareAddr
	revision [2]int
	features []string
	reset    bool
	closed   bool
}

func (p *fakeDeviceProbe) ChipName() string { return p.chip }
func (p *fakeDeviceProbe) FlashID() (uint8, uint16, error) {
	return 0xEF, p.flashID, p.flashErr
}
func (p *fakeDeviceProbe) MAC() (net.HardwareAddr, error) { return p.mac, nil }
func (p *fakeDeviceProbe) ChipRevision() (int, int, error) {
	return p.revision[0], p.revision[1], nil
}
func (p *fakeDeviceProbe) ChipFeatures() ([]string, error) { return p.features, nil }
func (p *fakeDeviceProbe) Reset()                          { p.reset = true }
func (p *fakeDeviceProbe) Close() error                    { p.closed = true; return nil }

func TestProbeReadsESPDeviceInformation(t *testing.T) {
	probe := &fakeDeviceProbe{
		chip:     "ESP32-S3",
		flashID:  0x4017,
		mac:      net.HardwareAddr{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
		revision: [2]int{1, 2},
		features: []string{"WiFi", "BLE"},
	}
	manager := ReleaseManager{OpenProbe: func(port string) (deviceProbe, error) {
		if port != "/dev/ttyUSB0" {
			t.Fatalf("unexpected port %q", port)
		}
		return probe, nil
	}}

	device, err := manager.Probe(context.Background(), Device{Port: "/dev/ttyUSB0"})
	if err != nil {
		t.Fatal(err)
	}
	want := &DeviceInfo{
		Chip: "ESP32-S3", Revision: "v1.2", FlashSize: "8 MB",
		FlashID: "EF:4017", MAC: "aa:bb:cc:dd:ee:ff", Features: []string{"WiFi", "BLE"},
	}
	if !reflect.DeepEqual(device.Info, want) {
		t.Fatalf("unexpected device info:\n got: %#v\nwant: %#v", device.Info, want)
	}
	if !probe.reset || !probe.closed {
		t.Fatalf("probe must reset and close the device: %+v", probe)
	}
}

func TestProbeKeepsPartialInformation(t *testing.T) {
	probe := &fakeDeviceProbe{chip: "ESP32-C3", flashErr: errors.New("unsupported")}
	manager := ReleaseManager{OpenProbe: func(string) (deviceProbe, error) { return probe, nil }}
	device, err := manager.Probe(context.Background(), Device{Port: "COM7"})
	if err != nil {
		t.Fatal(err)
	}
	if device.Info.Chip != "ESP32-C3" || len(device.Info.Warnings) != 1 || device.Info.FlashSize != "" {
		t.Fatalf("unexpected partial device info: %#v", device.Info)
	}
}

func TestFlashSizeFromID(t *testing.T) {
	for _, test := range []struct {
		id   uint16
		want string
	}{{0x4014, "1 MB"}, {0x4016, "4 MB"}, {0x4018, "16 MB"}, {0x4001, ""}} {
		if got := flashSizeFromID(test.id); got != test.want {
			t.Errorf("flashSizeFromID(%04X) = %q, want %q", test.id, got, test.want)
		}
	}
}
