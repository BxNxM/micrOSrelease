package tui

import tea "charm.land/bubbletea/v2"

func (m model) selfUpdateKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc", "ctrl+c", "q":
		if m.appUpdate.cancel != nil {
			m.appUpdate.cancel()
		}
		m.appUpdate.quitting = key != "esc"
		m.appUpdate.status = "Cancelling update…"
	}
	return m, nil
}

// toggleAppUpdateMode allows a session-only reinstall, intentionally absent from hints.
func (m model) toggleAppUpdateMode() (tea.Model, tea.Cmd) {
	if m.appUpdate.updater == nil {
		return m, nil
	}
	m.appUpdate.updateMode = !m.appUpdate.updateMode
	if !m.appUpdate.checking {
		if m.appUpdate.updateMode && m.appUpdate.offer.Version == "" {
			return m.requestAppUpdate()
		}
		m.appUpdate.status = ""
	}
	return m, nil
}
