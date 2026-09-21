package widgets

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// UpdateBannerLines shares the wrapped banner layout with grid pagination.
func (m State) UpdateBannerLines(width int) []string {
	if m.AppUpdateLine == "" {
		return nil
	}
	text := Clean(m.AppUpdateLine)
	release := lipgloss.NewStyle().Foreground(ColorRelease)
	plain := lipgloss.NewStyle().Foreground(ColorText)
	if prefix, ok := strings.CutSuffix(text, "(Press u to update)"); ok {
		text = release.Render(prefix) + plain.Render("(Press ") + release.Bold(true).Render("u") + plain.Render(" to update)")
	} else {
		text = release.Render(text)
	}
	return strings.Split(ansi.Wrap(text, max(1, width), ""), "\n")
}
