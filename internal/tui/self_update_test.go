package tui

import (
	"context"
	"errors"
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
	if updater.installed != 0 || !strings.Contains(m.appUpdateLine(), "u update") {
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

func TestHiddenUpdateBannerTogglePreservesUpdateState(t *testing.T) {
	updater := &fakeAppUpdater{}
	m := New(nil, nil, WithUpdater(updater, "0.1.0", "3.6.3-0"))
	if m.appUpdateLine() == "" {
		t.Fatal("banner initially hidden")
	}
	next, cmd := m.handleKey("x")
	m = next.(model)
	if cmd != nil || m.appUpdateLine() != "" || !m.appUpdate.checking {
		t.Fatal("toggle changed update operation")
	}
	next, _ = m.Update(updateCheckedMsg{offer: selfupdate.UpdateOffer{Version: "0.2.0", Available: true}})
	m = next.(model)
	if m.appUpdateLine() != "" || !m.appUpdate.offer.Available {
		t.Fatal("background result reset visibility or lost offer")
	}
	m.appUpdate.installing = true
	next, cmd = m.handleKey("x")
	m = next.(model)
	if cmd != nil || m.appUpdateLine() == "" || !m.appUpdate.installing || updater.installed != 0 {
		t.Fatal("toggle during update changed operation")
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
		if current.appUpdate.hidden {
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
