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

// toggleAppUpdateBanner is a session-only display test, intentionally absent from hints.
func (m model) toggleAppUpdateBanner() (tea.Model, tea.Cmd) {
	m.appUpdate.hidden = !m.appUpdate.hidden
	return m, nil
}
