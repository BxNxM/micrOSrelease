package widgets

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestCardTitleLockAlignment(t *testing.T) {
	for _, width := range []int{20, 37, 38} {
		for _, title := range []string{"node", StyleTitle.Render("Localhost · __simulator__ · 3.6.0")} {
			line := CardTitleWidget(width, title, "🔒")
			if lipgloss.Width(line) != width-4 || !strings.HasSuffix(line, "🔒") {
				t.Fatalf("lock alignment at width %d: %q", width, line)
			}
			card := CardWidget(width, 4, ColorBorder, line)
			if w, h := lipgloss.Size(card); w != width || h != 6 {
				t.Fatalf("card size = %dx%d", w, h)
			}
		}
	}
}

func TestCardDimensionsWithLongSpecialTitle(t *testing.T) {
	for _, width := range []int{20, 37, 38} {
		card := CardWidget(width, 4, ColorBorder,
			StyleTitle.Render("Localhost · __simulator__ · 3.6.0"),
			"127.0.0.1 · STA · 0.001s", "WEBUI: ON · ESPNOW: OFF", "CRON: ON · TIMIRQ: OFF")
		if gotWidth, gotHeight := lipgloss.Size(card); gotWidth != width || gotHeight != 6 {
			t.Fatalf("card size = %dx%d, want %dx6", gotWidth, gotHeight, width)
		}
	}
}
