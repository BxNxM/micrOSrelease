package views

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/micros/microsctl/internal/network"
	"github.com/micros/microsctl/internal/tui/widgets"
)

func TestNetworkActivityKeepsGridPosition(t *testing.T) {
	for _, nodes := range [][]network.Device{nil, {{Name: "test"}}} {
		state := widgets.State{Width: 90, Height: 30, Nodes: nodes}
		gridRow := -1
		for _, phase := range []string{"initial", "scanning", "updated"} {
			state.Discovering = phase == "scanning"
			want := "● Last updated: n/a"
			if state.Discovering {
				want = "● Background: Network scan & status refresh"
			} else if phase == "updated" {
				state.LastUpdated = time.Date(2026, 9, 20, 12, 34, 56, 0, time.Local)
				want = "● Last updated: 2026-09-20 12:34:56"
			}
			view := ansi.Strip(NodesView(state).Content)
			if !strings.Contains(view, want) {
				t.Fatalf("%s: missing status %q:\n%s", phase, want, view)
			}
			index := strings.Index(view, "USB Tools")
			if index < 0 {
				t.Fatal("USB Tools card missing")
			}
			row := strings.Count(view[:index], "\n")
			if gridRow < 0 {
				gridRow = row
			} else if row != gridRow {
				t.Fatalf("%s: grid moved from row %d to %d", phase, gridRow, row)
			}
		}
	}
}

func TestUpdateStatusHasStableTopRow(t *testing.T) {
	state := widgets.State{Width: 100, Height: 30}
	row := -1
	for _, line := range []string{"Checking for updates…", "u update & restart · microsctl 0.1.0 → 0.2.0", "Updating microsctl… 50%", "Update check unavailable · u retry"} {
		state.AppUpdateLine = line
		text := ansi.Strip(NodesView(state).Content)
		lines := strings.Split(text, "\n")
		if lines[1] != line {
			t.Fatalf("not on top status line: %s", text)
		}
		current := strings.Count(text[:strings.Index(text, "USB Tools")], "\n")
		if row >= 0 && row != current {
			t.Fatal("update status moved grid")
		}
		row = current
	}
}
