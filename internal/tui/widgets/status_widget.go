package widgets

import (
	"charm.land/lipgloss/v2"
	"strings"
)

func (m State) StatusWidget() string {
	color := ColorText
	if strings.Contains(strings.ToLower(m.Status), "failed") || strings.Contains(m.Status, "Select") {
		color = ColorError
	}
	return lipgloss.NewStyle().Foreground(color).Render("● " + m.Status)
}
