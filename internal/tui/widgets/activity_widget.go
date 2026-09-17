package widgets

import "charm.land/lipgloss/v2"

// ActivityWidget shows the currently active background operation.
func ActivityWidget(name string) string {
	if name == "" {
		return ""
	}
	return lipgloss.NewStyle().Foreground(ColorAccent).Render("● Background: " + name)
}
