package tui

import (
	"testing"

	"github.com/micros/microsctl/internal/usb"
)

func TestReturningAfterUSBOperationRefreshesNodes(t *testing.T) {
	for _, op := range []operation{operationInstall, operationUpdate} {
		for _, key := range []string{"esc", "backspace"} {
			for _, failed := range []bool{false, true} {
				d := gatedDiscovery{release: make(chan struct{})}
				close(d.release)
				m := model{network: d, operation: op}
				if failed {
					m.operationError = "upload failed"
				} else {
					m.result = &usb.Result{}
				}
				next, cmd := m.handleKey(key)
				m = next.(model)
				if !m.showNodes || !m.discovering || cmd == nil || m.result != nil || m.operationError != "" {
					t.Fatalf("return after %s did not start a clean Nodes refresh", op)
				}
				cancel := m.scanCancel
				// Drain the real streaming commands using a fake discoverer.
				for cmd != nil {
					next, cmd = m.Update(cmd())
					m = next.(model)
				}
				cancel()
				if m.discovering || len(m.nodes) != 2 {
					t.Fatal("refresh results did not reach Nodes")
				}
			}
		}
	}
}

func TestReturnRefreshUsesManualScanGuards(t *testing.T) {
	for _, removing := range []bool{false, true} {
		m := model{operation: operationInstall, result: &usb.Result{}, discovering: !removing}
		if removing {
			m.removingUID = "removing"
		}
		next, cmd := m.handleKey("esc")
		if cmd != nil || !next.(model).showNodes {
			t.Fatal("return overlapped an active scan/removal")
		}
		_, cmd = next.(model).handleKey("r")
		if cmd != nil {
			t.Fatal("manual and automatic refresh guards differ")
		}
	}
}

func TestReturningWithoutFinishedUSBWorkDoesNotScan(t *testing.T) {
	for _, m := range []model{{}, {operation: operationInstall}, {operation: operationUpdate, running: true}} {
		next, cmd := m.handleKey("esc")
		if cmd != nil || next.(model).discovering {
			t.Fatal("scanned before a USB operation finished")
		}
	}
}

func TestLeavingDuringUSBWorkRefreshesOnCompletion(t *testing.T) {
	m := model{operation: operationUpdate, running: true}
	next, cmd := m.handleKey("esc")
	m = next.(model)
	if cmd != nil || m.discovering || !m.running {
		t.Fatal("return must wait for USB work")
	}
	msg := operationMsg{done: true}
	next, cmd = m.handleOperation(msg)
	m = next.(model)
	if !m.discovering || cmd == nil || m.running || m.operationError != "" {
		t.Fatal("completion did not refresh Nodes")
	}
	m.scanCancel()
}
