package views

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/micros/microsctl/internal/network"
	"github.com/micros/microsctl/internal/tui/widgets"
)

func TestNodeAuthFeatures(t *testing.T) {
	for _, tc := range []struct {
		name, value, want string
	}{
		{"enabled", "ON", "ON"},
		{"disabled", "OFF", "OFF"},
		{"unknown", "", "n/a"},
	} {
		for _, width := range []int{0, 44, 85, 126} {
			t.Run(fmt.Sprintf("%s/width%d", tc.name, width), func(t *testing.T) {
				state := widgets.State{
					Width: width, Height: 30, NodeIndex: 1,
					Nodes: []network.Device{{Name: "test", Features: map[string]string{
						"auth": tc.value, "cron": "ON", "timirq": "OFF",
					}}},
				}
				cards := ansi.Strip(NodesView(state).Content)
				for _, want := range []string{
					"WEBUI: n/a · ESPNOW: n/a · AUTH: " + tc.want,
					"CRON: ON · TIMIRQ: OFF",
				} {
					if !strings.Contains(cards, want) {
						t.Fatalf("card missing %q:\n%s", want, cards)
					}
				}
				details := ansi.Strip(NodeDetailsView(state).Content)
				if !strings.Contains(details, fmt.Sprintf("%-10s %s", "AUTH", tc.want)) {
					t.Fatalf("details missing auth value %q:\n%s", tc.want, details)
				}
			})
		}
	}
}
