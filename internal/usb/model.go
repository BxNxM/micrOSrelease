package usb

import (
	"context"
	"io/fs"
	"time"
)

// Device is a USB serial endpoint that may host a supported microcontroller.
type Device struct {
	Port string
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
	Install(context.Context, Target, ...func([]Stage)) (Result, error)
	Update(context.Context, Target, ...func([]Stage)) (Result, error)
}

// DummyManager implements the prototype without touching hardware.
type DummyManager struct {
	Delay  time.Duration
	Assets fs.FS
}
