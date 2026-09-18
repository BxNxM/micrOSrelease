package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/micros/microsctl/internal/usb"
)

func TestReconnectStageExplainsWhenUnplugIsSafe(t *testing.T) {
	m := model{running: true, operation: operationUpdate, stages: []usb.Stage{{Name: "Reconnect to MicroPython REPL", State: usb.StageRunning, CanReconnect: true, Detail: "Waiting for the same board to return"}}}
	rendered := ansi.Strip(m.renderState().OperationWidget(90))
	if !strings.Contains(rendered, "unplug/replug is safe") || !strings.Contains(rendered, "Waiting for the same board") || strings.Contains(rendered, "Do not disconnect") {
		t.Fatalf("missing reconnect guidance: %s", rendered)
	}
	m.stages[0].State = usb.StageDone
	m.stages = append(m.stages, usb.Stage{Name: "Restore files", State: usb.StageRunning})
	rendered = ansi.Strip(m.renderState().OperationWidget(90))
	if strings.Contains(rendered, "unplug/replug is safe") || !strings.Contains(rendered, "Do not disconnect") {
		t.Fatal("unplug guidance leaked into file transfer stage")
	}
}

func TestOperationCancellationAndReturningPort(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	m := model{running: true, operationContext: ctx, operationCancel: cancel, inventory: usb.Inventory{Devices: []usb.Device{{Port: "old"}}}}
	next, _ := m.Update(operationMsg{done: true, result: usb.Result{Target: usb.Target{Device: usb.Device{Port: "new"}}}})
	got := next.(model)
	if got.inventory.Devices[0].Port != "new" || got.running || got.operationCancel != nil || ctx.Err() != context.Canceled {
		t.Fatal("completion did not release wait and update the selected port")
	}
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	m.operationContext, m.operationCancel = ctx, cancel
	m.handleKey("ctrl+c")
	if ctx.Err() != context.Canceled {
		t.Fatal("Ctrl+C did not cancel the USB wait")
	}
}

func TestDismissedOperationRetainsReturningPort(t *testing.T) {
	m := model{running: true, operation: operationUpdate, inventory: usb.Inventory{Devices: []usb.Device{{Port: "old"}}}}
	next, _ := m.handleKey("esc")
	m = next.(model)
	next, _ = m.Update(operationMsg{done: true, result: usb.Result{Target: usb.Target{Device: usb.Device{Port: "new"}}}})
	m = next.(model)
	if m.inventory.Devices[0].Port != "new" {
		t.Fatal("dismissed operation lost the returning device port")
	}
	if m.running || m.result != nil || len(m.stages) != 0 || m.status != "" {
		t.Fatal("dismissed operation restored transient feedback")
	}
}
