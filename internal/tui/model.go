package tui

import (
	"context"

	"github.com/micros/microsctl/internal/network"
	"github.com/micros/microsctl/internal/usb"
)

type action int

const (
	actionInstall action = iota
	actionUpdate
	actionRefresh
)

type operation string

const (
	operationInstall operation = "install"
	operationUpdate  operation = "update"
)

type model struct {
	usb     usb.Manager
	network network.Discoverer

	width  int
	height int
	cursor int

	inventory        usb.Inventory
	deviceIndex      int
	imageIndex       int
	firmwareSelected bool
	boardType        string
	firmwareIndex    int
	showFirmware     bool
	pendingOperation operation
	nodes            []network.Device
	showNodes        bool
	showNodeDetails  bool
	nodeIndex        int
	detailAction     int
	removingUID      string
	removedUID       string

	loadingInventory bool
	usbScanRequested bool
	usbScanned       bool
	discovering      bool
	scanCancel       context.CancelFunc
	confirming       bool
	running          bool
	operation        operation
	progress         int
	stages           []usb.Stage
	dismissOperation bool
	spinnerFrame     int

	status string
	result *usb.Result
}

var actions = []struct {
	title       string
	description string
}{
	{"Install micrOS", "Clean USB install; erases the selected device"},
	{"Update micrOS", "USB update; preserves the node configuration"},
	{"USB Scan", "Find USB devices"},
}

// New creates the UI and injects all feature implementations.
func New(usbManager usb.Manager, networkDiscoverer network.Discoverer) model {
	return model{
		usb:              usbManager,
		network:          networkDiscoverer,
		loadingInventory: true,
		showNodes:        true,
		status:           "Loading release environment…",
	}
}
