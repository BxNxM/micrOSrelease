package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/micros/microsctl/internal/usb"
)

// clearOperation discards transient feedback without changing the release target.
func (m *model) clearOperation() {
	m.result = nil
	m.stages = nil
	m.progress = 0
	m.spinnerFrame = 0
	m.operation = ""
	m.status = ""
}

func (m model) handleOperation(msg operationMsg) (tea.Model, tea.Cmd) {
	if !msg.done {
		m.stages = msg.stages
		completed := 0
		for _, stage := range m.stages {
			if stage.State == usb.StageDone || stage.State == usb.StageSkipped {
				completed++
			}
		}
		m.progress = completed * 100 / max(1, len(m.stages))
		return m, nextOperation(msg.queue)
	}
	m.running = false
	if m.operationCancel != nil {
		m.operationCancel()
		m.operationCancel = nil
		m.operationContext = nil
	}
	// Keep the returning endpoint even when the operation card was dismissed.
	if msg.err == nil && msg.result.Target.Device.Port != "" && m.deviceIndex >= 0 && m.deviceIndex < len(m.inventory.Devices) {
		m.inventory.Devices[m.deviceIndex] = msg.result.Target.Device
	}
	if m.dismissOperation {
		m.clearOperation()
		return m, nil
	}
	if msg.err != nil {
		m.status = "Operation failed: " + msg.err.Error()
		return m, nil
	}
	m.result = &msg.result
	m.progress = 100
	m.status = msg.result.Summary
	return m, nil
}
