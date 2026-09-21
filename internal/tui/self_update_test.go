package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	selfupdate "github.com/micros/microsctl/internal/network/self_update"
)

type fakeAppUpdater struct {
	installed int
	failure   error
}

func (f *fakeAppUpdater) Check(context.Context) (selfupdate.UpdateOffer, error) {
	return selfupdate.UpdateOffer{Version: "0.2.0", MicrOSVersion: "3.6.3-0", Available: true}, nil
}
func (f *fakeAppUpdater) Install(ctx context.Context, offer selfupdate.UpdateOffer, emit func(int)) (string, error) {
	f.installed++
	emit(50)
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	return "backup", f.failure
}

func TestSelfUpdateNeedsUAndRestartsAfterSuccess(t *testing.T) {
	updater := &fakeAppUpdater{}
	m := New(nil, nil, WithUpdater(updater, "0.1.0", "3.6.3-0"))
	m.loadingInventory = false
	next, _ := m.Update(m.checkAppUpdateCmd()())
	m = next.(model)
	if updater.installed != 0 || !strings.Contains(m.appUpdateLine(), "Update 0.2.0 · Press u to update") {
		t.Fatal("check did not offer update")
	}
	next, cmd := m.handleKey("u")
	m = next.(model)
	if !m.appUpdate.installing || m.RestartRequested() {
		t.Fatal("update not started or restarted too early")
	}
	for cmd != nil {
		msg := cmd()
		if _, quit := msg.(tea.QuitMsg); quit {
			break
		}
		next, cmd = m.Update(msg)
		m = next.(model)
	}
	if updater.installed != 1 || !m.RestartRequested() {
		t.Fatal("update did not restart")
	}
}

func TestFailedSelfUpdateDoesNotRestart(t *testing.T) {
	updater := &fakeAppUpdater{failure: errors.New("download failed")}
	m := New(nil, nil, WithUpdater(updater, "0.1.0", "3.6.3-0"))
	m.loadingInventory = false
	next, _ := m.Update(m.checkAppUpdateCmd()())
	m = next.(model)
	next, cmd := m.handleKey("u")
	m = next.(model)
	for cmd != nil {
		next, cmd = m.Update(cmd())
		m = next.(model)
	}
	if m.RestartRequested() || m.appUpdate.installing || !strings.Contains(m.appUpdateLine(), "download failed") {
		t.Fatal(m.appUpdateLine())
	}
}

func TestSelfUpdateWaitsForUSBandCancellationCompletion(t *testing.T) {
	updater := &fakeAppUpdater{}
	m := New(nil, nil, WithUpdater(updater, "0.1.0", "3.6.3-0"))
	m.loadingInventory = false
	next, _ := m.Update(m.checkAppUpdateCmd()())
	m = next.(model)
	m.running = true
	next, cmd := m.handleKey("u")
	m = next.(model)
	if cmd != nil || m.appUpdate.installing {
		t.Fatal("update allowed during USB work")
	}
	m.running = false
	next, cmd = m.handleKey("u")
	m = next.(model)
	if cmd == nil {
		t.Fatal("update not started")
	}
	ctxCancelled := false
	oldCancel := m.appUpdate.cancel
	m.appUpdate.cancel = func() { ctxCancelled = true; oldCancel() }
	next, quitCmd := m.handleKey("q")
	m = next.(model)
	if quitCmd != nil || !ctxCancelled || !m.appUpdate.installing {
		t.Fatal("quit did not wait for update cancellation")
	}
	for cmd != nil {
		msg := cmd()
		if _, quit := msg.(tea.QuitMsg); quit {
			break
		}
		next, cmd = m.Update(msg)
		m = next.(model)
	}
	if m.RestartRequested() || m.appUpdate.installing {
		t.Fatal("cancelled update restarted")
	}
}

func TestUpdateCheckFailureCanRetryWithoutBlockingNodes(t *testing.T) {
	m := New(nil, nil, WithUpdater(&fakeAppUpdater{}, "0.1.0", "3.6.3-0"))
	next, _ := m.Update(updateCheckedMsg{err: errors.New("offline")})
	m = next.(model)
	if m.appUpdate.checking || m.appUpdate.offer.Available {
		t.Fatal("bad offline state")
	}
	next, cmd := m.handleKey("u")
	m = next.(model)
	if cmd == nil || !m.appUpdate.checking {
		t.Fatal("cannot retry check")
	}
}

func TestUpdateModeToggleAndSameVersionInstall(t *testing.T) {
	for _, available := range []bool{false, true} {
		t.Run(fmt.Sprint(available), func(t *testing.T) {
			updater := &fakeAppUpdater{}
			current := "0.2.0"
			if available {
				current = "0.1.0"
			}
			m := New(nil, nil, WithUpdater(updater, current, "3.6.3-0"))
			m.loadingInventory = false
			next, _ := m.Update(updateCheckedMsg{offer: selfupdate.UpdateOffer{Version: "0.2.0", Available: available}})
			m = next.(model)
			base := "microsctl " + current + " · micrOS 3.6.3-0"
			extended := base + " · Update 0.2.0 · Press u to update"
			want := base
			if available {
				want = extended
			}
			if m.appUpdateLine() != want || updater.installed != 0 {
				t.Fatalf("unexpected initial banner: %s", m.appUpdateLine())
			}
			// X toggles the offer without hiding the version line or installing.
			for range 2 {
				next, cmd := m.handleKey("x")
				m = next.(model)
				if want == base {
					want = extended
				} else {
					want = base
				}
				if cmd != nil || m.appUpdateLine() != want || updater.installed != 0 {
					t.Fatalf("unexpected toggled banner: %s", m.appUpdateLine())
				}
			}
			if !m.appUpdate.updateMode {
				next, _ = m.handleKey("x")
				m = next.(model)
			}
			next, cmd := m.handleKey("u")
			m = next.(model)
			if !m.appUpdate.installing || cmd == nil {
				t.Fatal("active update mode did not start installation")
			}
			next, toggleCmd := m.handleKey("x")
			m = next.(model)
			if toggleCmd != nil || !m.appUpdate.updateMode || !strings.Contains(m.appUpdateLine(), "Updating microsctl") {
				t.Fatal("x changed an active installation")
			}
			for cmd != nil {
				msg := cmd()
				if _, quit := msg.(tea.QuitMsg); quit {
					break
				}
				next, cmd = m.Update(msg)
				m = next.(model)
			}
			if updater.installed != 1 || !m.RestartRequested() {
				t.Fatal("installation did not complete and request restart")
			}
		})
	}
}

func TestManualUpdateModeSurvivesPendingCheck(t *testing.T) {
	m := New(nil, nil, WithUpdater(&fakeAppUpdater{}, "0.2.0", "3.6.3-0"))
	next, cmd := m.handleKey("x")
	m = next.(model)
	if cmd != nil || !m.appUpdate.checking || !m.appUpdate.updateMode {
		t.Fatal("toggle disrupted pending check")
	}
	next, _ = m.Update(updateCheckedMsg{offer: selfupdate.UpdateOffer{Version: "0.2.0"}})
	m = next.(model)
	if !strings.Contains(m.appUpdateLine(), "Update 0.2.0 · Press u to update") {
		t.Fatal("check lost manual update mode")
	}
}

func TestManualUpdateModeRequiresSuccessfulCheck(t *testing.T) {
	updater := &fakeAppUpdater{}
	m := New(nil, nil, WithUpdater(updater, "0.2.0", "3.6.3-0"))
	next, _ := m.Update(updateCheckedMsg{err: errors.New("offline")})
	m = next.(model)
	next, cmd := m.handleKey("x")
	m = next.(model)
	if cmd == nil || !m.appUpdate.checking || !m.appUpdate.updateMode || m.appUpdate.installing || updater.installed != 0 {
		t.Fatal("manual mode must fetch a valid offer before installing")
	}
}

func TestBannerToggleOnlyOnNodesAndNotInHints(t *testing.T) {
	m := New(nil, nil, WithUpdater(&fakeAppUpdater{}, "0.1.0", "3.6.3-0"))
	for _, screen := range []string{"details", "firmware", "usb", "shell"} {
		current := m
		switch screen {
		case "details":
			current.showNodeDetails = true
		case "firmware":
			current.showFirmware = true
		case "usb":
			current.showNodes = false
		case "shell":
			current.shell.visible = true
		}
		next, _ := current.handleKey("x")
		current = next.(model)
		if current.appUpdate.updateMode {
			t.Fatalf("x toggled banner on %s", screen)
		}
		if screen == "shell" && current.shell.input != "x" {
			t.Fatal("shell x input intercepted")
		}
	}
	text := m.View().Content
	if strings.Contains(text, "x hide") || strings.Contains(text, "x show") || strings.Contains(text, "x toggle") {
		t.Fatal("test key added to hints")
	}
}
