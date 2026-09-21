package tui

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	selfupdate "github.com/micros/microsctl/internal/network/self_update"
)

type appUpdateState struct {
	updateMode                              bool
	updater                                 selfupdate.AppUpdater
	current, firmware, status               string
	offer                                   selfupdate.UpdateOffer
	checking, installing, restart, quitting bool
	cancel                                  context.CancelFunc
}

type Option func(*model)

func WithUpdater(updater selfupdate.AppUpdater, version, firmware string) Option {
	return func(m *model) {
		m.appUpdate = appUpdateState{updater: updater, current: version, firmware: firmware, checking: true, status: "Checking for updates…"}
	}
}

type updateCheckedMsg struct {
	offer selfupdate.UpdateOffer
	err   error
}
type appUpdateMsg struct {
	percentage int
	done       bool
	backup     string
	err        error
	queue      <-chan appUpdateMsg
}

// RestartRequested is inspected only after the TUI has restored the terminal.
func (m model) RestartRequested() bool { return m.appUpdate.restart }

func (m model) checkAppUpdateCmd() tea.Cmd {
	if m.appUpdate.updater == nil {
		return nil
	}
	return func() tea.Msg {
		offer, err := m.appUpdate.updater.Check(context.Background())
		return updateCheckedMsg{offer: offer, err: err}
	}
}

func (m model) handleUpdateChecked(msg updateCheckedMsg) (tea.Model, tea.Cmd) {
	m.appUpdate.checking = false
	m.appUpdate.offer = msg.offer
	switch {
	case msg.err != nil:
		m.appUpdate.status = "Update check unavailable · u retry"
	default:
		m.appUpdate.updateMode = m.appUpdate.updateMode || msg.offer.Available
		m.appUpdate.status = ""
	}
	return m, nil
}

func (m model) requestAppUpdate() (tea.Model, tea.Cmd) {
	if m.appUpdate.updater == nil || m.appUpdate.checking || m.appUpdate.installing {
		return m, nil
	}
	if !m.appUpdate.updateMode || m.appUpdate.offer.Version == "" {
		m.appUpdate.checking = true
		m.appUpdate.status = "Checking for updates…"
		return m, m.checkAppUpdateCmd()
	}
	if m.running || m.probingUSB || m.loadingInventory {
		m.appUpdate.status = "Update available · wait for USB work to finish, then press u"
		return m, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.appUpdate.cancel = cancel
	m.appUpdate.installing = true
	m.appUpdate.status = "Updating microsctl… 0% · esc cancel"
	updater, offer := m.appUpdate.updater, m.appUpdate.offer
	queue := make(chan appUpdateMsg, 8)
	return m, func() tea.Msg {
		go func() {
			defer close(queue)
			backup, err := updater.Install(ctx, offer, func(percentage int) {
				select {
				case queue <- appUpdateMsg{percentage: percentage}:
				case <-ctx.Done():
				}
			})
			// Always report completion, including cancellation, before the app may exit.
			queue <- appUpdateMsg{done: true, backup: backup, err: err}
		}()
		return nextAppUpdate(queue)()
	}
}

func nextAppUpdate(queue <-chan appUpdateMsg) tea.Cmd {
	return func() tea.Msg { msg := <-queue; msg.queue = queue; return msg }
}

func (m model) handleAppUpdate(msg appUpdateMsg) (tea.Model, tea.Cmd) {
	if !msg.done {
		m.appUpdate.status = fmt.Sprintf("Updating microsctl… %d%% · esc cancel", msg.percentage)
		return m, nextAppUpdate(msg.queue)
	}
	m.appUpdate.installing = false
	if m.appUpdate.cancel != nil {
		m.appUpdate.cancel()
		m.appUpdate.cancel = nil
	}
	if msg.err != nil {
		m.appUpdate.status = "Update failed: " + msg.err.Error() + " · u retry"
	} else {
		m.appUpdate.status = "Updated · restarting…"
		m.appUpdate.restart = !m.appUpdate.quitting
	}
	if m.appUpdate.restart || m.appUpdate.quitting {
		if m.scanCancel != nil {
			m.scanCancel()
		}
		return m, tea.Quit
	}
	return m, nil
}

func (m model) appUpdateLine() string {
	if m.appUpdate.updater == nil {
		return ""
	}
	line := fmt.Sprintf("microsctl %s · micrOS %s", m.appUpdate.current, m.appUpdate.firmware)
	if m.appUpdate.status != "" {
		return line + " · " + m.appUpdate.status
	}
	if m.appUpdate.updateMode && m.appUpdate.offer.Version != "" {
		return line + fmt.Sprintf(" · Update available %s (Press u to update)", m.appUpdate.offer.Version)
	}
	return line
}
