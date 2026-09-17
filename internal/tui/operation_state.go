package tui

// clearOperation discards transient feedback without changing the release target.
func (m *model) clearOperation() {
	m.result = nil
	m.stages = nil
	m.progress = 0
	m.spinnerFrame = 0
	m.operation = ""
	m.status = ""
}
