package usb

import (
	"context"
	"io/fs"
)

// Device is a USB serial endpoint that may host a supported microcontroller.
type Device struct {
	Port      string
	Info      *DeviceInfo
	USBSerial string
	USBVID    string
	USBPID    string

	// USBLocation is the physical USB connection, when provided by the host.
	USBLocation string
}

// DeviceInfo contains optional metadata read from an ESP ROM bootloader.
// A nil Info means that the serial endpoint has not been identified yet.
type DeviceInfo struct {
	Chip      string
	Revision  string
	FlashSize string
	FlashID   string
	MAC       string
	Features  []string
	Warnings  []string
}

// Image is a firmware image and its target board type.
type Image struct {
	Path        string
	Name        string
	Board       string
	Release     bool
	Size        int64
	MicroPython string
	Version     string
	InstallHint string
}

// Inventory contains locally connected USB ports and available firmware.
type Inventory struct {
	Devices []Device
	Images  []Image
}

// Target fully identifies a USB release operation.
type Target struct {
	Device Device
	Image  Image
}

// Result describes a completed release operation.
type Result struct {
	Operation string
	Target    Target
	Summary   string
	Log       []string
}

// Manager is the UI-independent USB release API.
type Manager interface {
	Inventory(context.Context) (Inventory, error)
	Probe(context.Context, Device) (Device, error)
	Install(context.Context, Target, ...func([]Stage)) (Result, error)
	Update(context.Context, Target, ...func([]Stage)) (Result, error)
}

// ReleaseManager discovers USB endpoints and installs or updates firmware.
type ReleaseManager struct {
	Assets    fs.FS
	BackupDir string
	// Discover is optional and primarily supports deterministic callers and tests.
	Discover func(context.Context) ([]Device, error)
	// OpenFlasher is optional and keeps hardware access replaceable in tests.
	OpenFlasher func(string, InstallConfig) (deviceFlasher, error)
	// OpenProbe is optional and keeps device identification replaceable in tests.
	OpenProbe func(string) (deviceProbe, error)
	// OpenREPL is optional and keeps MicroPython serial access replaceable in tests.
	OpenREPL func(context.Context, string, REPLConfig) (replSession, error)
}
