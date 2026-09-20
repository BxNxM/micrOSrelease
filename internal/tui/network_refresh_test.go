package tui

import (
	"errors"
	"testing"
	"time"
)

func TestNetworkRefreshTimestamp(t *testing.T) {
	previous := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	for _, scanErr := range []error{nil, errors.New("scan failed")} {
		m := model{discovering: true, lastUpdated: previous}
		before := time.Now()
		updated, _ := m.Update(networkMsg{done: true, err: scanErr})
		m = updated.(model)
		if m.discovering {
			t.Fatal("completed scan still marked active")
		}
		if scanErr != nil {
			if !m.lastUpdated.Equal(previous) {
				t.Fatal("failed scan replaced last successful refresh time")
			}
		} else if m.lastUpdated.Before(before) || m.lastUpdated.After(time.Now()) {
			t.Fatalf("refresh time was not recorded at completion: %v", m.lastUpdated)
		}
		if !m.renderState().LastUpdated.Equal(m.lastUpdated) {
			t.Fatal("rendering snapshot lost refresh time")
		}
	}
}
