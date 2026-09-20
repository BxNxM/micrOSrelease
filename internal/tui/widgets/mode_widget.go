package widgets

import "charm.land/lipgloss/v2"

// ModeWidget gives runtime modes consistent colors across cards and details.
func ModeWidget(mode string) string {
	color := ColorMuted
	switch mode {
	case "dev":
		color = ColorOff
	case "rel":
		color = ColorRelease
	}
	return lipgloss.NewStyle().Bold(false).Foreground(color).Render(Clean(mode))
}
