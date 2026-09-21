package tui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/micros/microsctl/internal/network"
	"github.com/micros/microsctl/internal/usb"
)

type inventoryMsg struct {
	inventory usb.Inventory
	err       error
}

type usbProbeMsg struct {
	index  int
	device usb.Device
	err    error
}

type networkMsg struct {
	device *network.Device
	done   bool
	err    error
	queue  <-chan networkMsg
}

type operationMsg struct {
	result usb.Result
	err    error
	stages []usb.Stage
	done   bool
	queue  <-chan operationMsg
}

type tickMsg time.Time
type startScanMsg struct{}
type autoRefreshMsg struct{}

const autoRefreshInterval = 5 * time.Minute

func autoRefreshCmd() tea.Cmd {
	return tea.Tick(autoRefreshInterval, func(time.Time) tea.Msg { return autoRefreshMsg{} })
}

type cachedNodesMsg struct {
	nodes []network.Device
	err   error
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.inventoryCmd(), autoRefreshCmd(), m.checkAppUpdateCmd(), func() tea.Msg {
		if source, ok := m.network.(interface {
			Cached() ([]network.Device, error)
		}); ok {
			nodes, err := source.Cached()
			return cachedNodesMsg{nodes: nodes, err: err}
		}
		return startScanMsg{}
	})
}

func (m model) inventoryCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		inventory, err := m.usb.Inventory(ctx)
		return inventoryMsg{inventory: inventory, err: err}
	}
}

func (m model) probeUSBCmd(index int) tea.Cmd {
	device := m.inventory.Devices[index]
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		identified, err := m.usb.Probe(ctx, device)
		return usbProbeMsg{index: index, device: identified, err: err}
	}
}

func (m *model) networkCmd() tea.Cmd {
	m.removedUID = ""
	// Retain identity between scans, but do not prefer last scan's reachable
	// alias over a fresh unavailable special endpoint observation.
	for i := range m.nodeObservations {
		m.nodeObservations[i].Cached = true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	m.scanCancel = cancel
	queue := make(chan networkMsg, 32)
	discoverer := m.network
	return func() tea.Msg {
		go func() {
			defer cancel()
			defer close(queue)
			err := discoverer.Discover(ctx, func(device network.Device) {
				select {
				case queue <- networkMsg{device: &device}:
				case <-ctx.Done():
				}
			})
			select {
			case queue <- networkMsg{done: true, err: err}:
			case <-ctx.Done():
			}
		}()
		return nextNode(queue)()
	}
}

func nextNode(queue <-chan networkMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-queue
		if !ok {
			return networkMsg{done: true}
		}
		msg.queue = queue
		return msg
	}
}

func (m model) operationCmd() tea.Cmd {
	target := m.selectedTarget()
	requested := m.operation
	queue := make(chan operationMsg, 32)
	return func() tea.Msg {
		go func() {
			defer close(queue)
			ctx := m.operationContext
			if ctx == nil {
				ctx = context.Background()
			}
			emit := func(stages []usb.Stage) {
				select {
				case queue <- operationMsg{stages: stages}:
				case <-ctx.Done():
				}
			}
			var (
				result usb.Result
				err    error
			)
			switch requested {
			case operationInstall:
				result, err = m.usb.Install(ctx, target, emit)
			case operationUpdate:
				result, err = m.usb.Update(ctx, target, emit)
			}
			queue <- operationMsg{result: result, err: err, done: true}
		}()
		return nextOperation(queue)()
	}
}

func nextOperation(queue <-chan operationMsg) tea.Cmd {
	return func() tea.Msg {
		msg := <-queue
		msg.queue = queue
		return msg
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(90*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m model) selectedTarget() usb.Target {
	if !m.firmwareSelected || len(m.inventory.Devices) == 0 || len(m.inventory.Images) == 0 {
		return usb.Target{}
	}
	return usb.Target{
		Device: m.inventory.Devices[m.deviceIndex],
		Image:  m.inventory.Images[m.imageIndex],
	}
}
