package tui

import (
	"github.com/micros/microsctl/internal/usb"
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

func TestHiddenOperationCompletionDoesNotRestoreCard(t *testing.T) {
	m := model{running: true, operation: operationInstall}
	next, _ := m.handleKey("esc")
	m = next.(model)
	next, _ = m.Update(operationMsg{done: true, result: usb.Result{Summary: "Completed"}})
	m = next.(model)
	if m.running || m.result != nil || len(m.stages) != 0 || m.status != "" {
		t.Fatal("completion after leaving must not restore stale feedback")
	}
}
