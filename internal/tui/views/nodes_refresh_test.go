package views

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
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
	for _, line := range []string{"Checking for updates…", "microsctl 0.1.0 · Update available 0.2.0 (Press u to update)", "Updating microsctl… 50%", "Update check unavailable · u retry"} {
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

func TestUpdateBannerWrapsWithoutLosingTextAndReservesGridSpace(t *testing.T) {
	const banner = "microsctl 1.0.1 · micrOS 3.6.3-0 · Update available 1.0.0 (Press u to update)"
	state := widgets.State{Height: 23, AppUpdateLine: banner, Nodes: []network.Device{{Name: "first"}, {Name: "second"}}}
	// Resize the same snapshot in both directions to verify pagination recovers.
	for _, terminalWidth := range []int{120, 44, 70, 120} {
		state.Width = terminalWidth
		content := NodesView(state).Content
		key := lipgloss.NewStyle().Bold(true).Foreground(widgets.ColorRelease).Render("u")
		if !strings.Contains(content, key) {
			t.Fatalf("update key is not bold at width %d: %q", terminalWidth, content)
		}
		text := ansi.Strip(content)
		start := strings.Index(text, "\n") + 1
		end := strings.Index(text, "TCP 9008")
		if end < start {
			t.Fatalf("missing banner at width %d: %s", terminalWidth, text)
		}
		rendered := strings.TrimSuffix(text[start:end], "\n")
		if strings.Join(strings.Fields(rendered), " ") != banner {
			t.Fatalf("banner text lost at width %d: %q", terminalWidth, rendered)
		}
		width, _, rows := state.NodeGrid()
		for _, line := range strings.Split(rendered, "\n") {
			if ansi.StringWidth(line) > width {
				t.Fatalf("banner exceeds width %d: %q", width, line)
			}
		}
		if terminalWidth == 120 {
			plain := lipgloss.NewStyle().Foreground(widgets.ColorText)
			for _, part := range []string{"(Press ", " to update)"} {
				if !strings.Contains(content, plain.Render(part)) {
					t.Fatalf("update instruction is not white: %q", content)
				}
			}
			if strings.Contains(rendered, "\n") || rows != 2 {
				t.Fatalf("wide layout did not recover: banner=%q rows=%d", rendered, rows)
			}
		} else if !strings.Contains(rendered, "\n") || rows != 1 {
			t.Fatalf("wrapped banner did not reserve space: banner=%q rows=%d", rendered, rows)
		}
		if terminalWidth == 44 && !strings.Contains(text, "page 1/3") {
			t.Fatalf("narrow layout pagination is incorrect: %s", text)
		}
	}
}
