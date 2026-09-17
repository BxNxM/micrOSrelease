package widgets

import "charm.land/lipgloss/v2"

var FeatureKeys = []string{"webui", "espnow", "cron", "timirq"}

// FeatureValueWidget is shared by node cards and the device details screen.
func FeatureValueWidget(value string) string {
	color := ColorMuted
	switch value {
	case "ON":
		color = ColorOnline
	case "OFF":
		color = ColorOff
	default:
		value = "n/a"
	}
	return lipgloss.NewStyle().Bold(false).Foreground(color).Render(value)
}
