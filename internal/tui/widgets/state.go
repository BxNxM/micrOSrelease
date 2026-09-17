package widgets

import (
	"github.com/micros/micros-release/internal/network"
	"github.com/micros/micros-release/internal/usb"
)

// State is a read-only rendering snapshot. Widgets never perform I/O or update
// application state; the tui package owns those responsibilities.
type State struct {
	Width, Height, Cursor                          int
	Inventory                                      usb.Inventory
	DeviceIndex, ImageIndex, FirmwareIndex         int
	FirmwareSelected                               bool
	BoardType                                      string
	Nodes                                          []network.Device
	NodeIndex                                      int
	DetailAction                                   int
	LoadingInventory, UsbScanRequested, UsbScanned bool
	Discovering, Confirming, Running               bool
	Operation                                      string
	Progress, SpinnerFrame                         int
	Stages                                         []usb.Stage
	Status                                         string
	Result                                         *usb.Result
	Target                                         usb.Target
	Actions                                        []string
}
