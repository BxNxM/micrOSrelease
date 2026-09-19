package tui

import (
	"errors"
	"github.com/micros/microsctl/internal/usb"
	"strings"
	"testing"
)

func TestLeavingActionsClearsOperation(t *testing.T) {
	for _, key := range []string{"esc", "backspace"} {
		m := model{result: &usb.Result{}, stages: []usb.Stage{{State: usb.StageDone}}, progress: 100, status: "Completed"}
		next, _ := m.handleKey(key)
		m = next.(model)
		if !m.showNodes || m.result != nil || len(m.stages) != 0 || m.progress != 0 || m.status != "" {
			t.Fatal("leaving Actions should clear stale feedback")
		}
	}
}

func TestOperationFailureStaysInsideInstallOrUpdateCard(t *testing.T) {
	for _, op := range []operation{operationInstall, operationUpdate} {
		for _, hasStage := range []bool{false, true} {
			m := model{running: true, operation: op, width: 100}
			if hasStage {
				m.stages = []usb.Stage{{Name: "Copy resources", State: usb.StageFailed}}
			}
			reason := "copy resource /modules/example.mpy: verification failed (device backup: /backups/device.zip)"
			next, _ := m.handleOperation(operationMsg{done: true, err: errors.New(reason)})
			m = next.(model)
			if m.running || m.operationError != reason || m.result != nil {
				t.Fatalf("failure not retained: %+v", m)
			}
			view := m.View().Content
			if !strings.Contains(view, strings.ToUpper(string(op))) || !strings.Contains(view, "Error:") || !strings.Contains(view, "example.mpy") || !strings.Contains(view, "device.zip") {
				t.Fatalf("failure missing from operation card: %s", view)
			}
			m.clearOperation()
			if m.operationError != "" {
				t.Fatal("old failure leaked after leaving operation")
			}
		}
	}
}

func TestNewOperationClearsPreviousFailure(t *testing.T) {
	m := model{operationError: "old error", stages: []usb.Stage{{State: usb.StageFailed}}, firmwareSelected: true,
		inventory: usb.Inventory{Devices: []usb.Device{{Port: "test"}}, Images: []usb.Image{{Path: "test.bin"}}}}
	next, _ := m.confirm(operationUpdate)
	m = next.(model)
	if m.operationError != "" || len(m.stages) != 0 || !m.confirming {
		t.Fatal("new operation retained stale failure")
	}
}

func TestHiddenOperationCompletionDoesNotRestoreCard(t *testing.T) {
	m := model{running: true, operation: operationInstall}
	next, _ := m.handleKey("esc")
	m = next.(model)
	next, cmd := m.Update(operationMsg{done: true, result: usb.Result{Summary: "Completed"}})
	m = next.(model)
	defer m.scanCancel()
	if m.running || m.result != nil || len(m.stages) != 0 || !m.discovering || cmd == nil {
		t.Fatal("completion after leaving must not restore stale feedback")
	}
}

func TestBackgroundFailureRetainsRecoveryDetails(t *testing.T) {
	for _, reopen := range []bool{false, true} {
		m := model{running: true, operation: operationUpdate, width: 100}
		next, _ := m.handleKey("esc")
		m = next.(model)
		if reopen {
			next, _ = m.handleKey("enter")
			m = next.(model)
		}
		reason := "restore failed (device backup: /backups/recovery.zip)"
		next, cmd := m.handleOperation(operationMsg{done: true, err: errors.New(reason)})
		m = next.(model)
		if m.running || m.showNodes || m.dismissOperation || m.operationError != reason || cmd != nil {
			t.Fatalf("background failure was dismissed: %+v", m)
		}
		if view := m.View().Content; !strings.Contains(view, "recovery.zip") {
			t.Fatalf("recovery details missing: %s", view)
		}
		next, _ = m.handleKey("esc")
		m = next.(model)
		if m.operationError != "" || !m.showNodes {
			t.Fatal("acknowledged failure should clear on leaving the panel")
		}
		m.scanCancel()
	}
}
