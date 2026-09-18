package widgets

import (
	"testing"

	"charm.land/lipgloss/v2"
)

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
