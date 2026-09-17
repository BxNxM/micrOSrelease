package network

import (
	"context"
	"time"
)

// Device is a micrOS node discovered through the release service port.
type Device struct {
	Name      string
	Address   string
	UID       string
	Mode      string
	Online    bool
	Version   string
	Latency   time.Duration
	Features  map[string]string
	Error     string
	CheckedAt time.Time
	Cached    bool `json:"-"`
}

type DeviceStore interface {
	LoadDevices() ([]Device, error)
	SaveDevices([]Device) error
}

// Discoverer is the UI-independent network discovery feature.
type Discoverer interface {
	Discover(context.Context, func(Device)) error
}
