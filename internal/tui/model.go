package tui

import (
	"context"
	"time"

	"github.com/micros/microsctl/internal/network"
	"github.com/micros/microsctl/internal/usb"
)

type action int

const (
	actionDiscovery action = iota
	actionInstall
	actionUpdate
	actionRefresh
)

type operation string

const (
	operationInstall operation = "install"
	operationUpdate  operation = "update"
)

type model struct {
	appUpdate appUpdateState
	shell     shellState
	usb       usb.Manager
	network   network.Discoverer

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
	nodeObservations []network.Device
	showNodes        bool
	showNodeDetails  bool
	nodeIndex        int
	detailAction     int
	removingUID      string
	removedUID       string

	loadingInventory      bool
	probingUSB            bool
	usbScanRequested      bool
	usbScanned            bool
	usbDiscoveryRequested bool
	usbDiscoveryFailures  int
	discovering           bool
	lastUpdated           time.Time
	scanCancel            context.CancelFunc
	confirming            bool
	operationContext      context.Context
	operationCancel       context.CancelFunc
	running               bool
	operation             operation
	operationError        string
	progress              int
	stages                []usb.Stage
	dismissOperation      bool
	spinnerFrame          int

	status string
	result *usb.Result
}

var actions = []struct {
	title       string
	description string
}{
	{"Discovery", "Scan and identify USB devices; selects matching framework · resets boards"},
	{"Install micrOS", "Clean USB install; erases the selected device"},
	{"Update micrOS", "USB update; preserves the node configuration"},
	{"USB Scan", "Find USB devices"},
}

// New creates the UI and injects all feature implementations.
func New(usbManager usb.Manager, networkDiscoverer network.Discoverer, options ...Option) model {
	m := model{
		usb:              usbManager,
		network:          networkDiscoverer,
		loadingInventory: true,
		showNodes:        true,
		status:           "Loading release environment…",
	}
	for _, option := range options {
		option(&m)
	}
	return m
}
