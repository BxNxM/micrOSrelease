package widgets

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// CardTitleWidget reserves the right edge of the content line for an indicator.
func CardTitleWidget(width int, title, indicator string) string {
	available := max(1, width-4)
	if indicator == "" {
		return ansi.Truncate(title, available, "…")
	}
	title = ansi.Truncate(title, max(0, available-lipgloss.Width(indicator)-1), "…")
	return title + strings.Repeat(" ", max(0, available-lipgloss.Width(title)-lipgloss.Width(indicator))) + indicator
}

// CardWidget renders a bounded card while preserving styles in its content.
func CardWidget(width, height int, border color.Color, lines ...string) string {
	content := make([]string, len(lines))
	for i, line := range lines {
		content[i] = ansi.Truncate(line, max(1, width-4), "…")
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(border).
		Padding(0, 1).Width(width).Height(height + 2).Render(strings.Join(content, "\n"))
}
