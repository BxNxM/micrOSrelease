package widgets

import (
	"charm.land/lipgloss/v2"
	"fmt"
	"strings"
)

func (m State) ConfirmationWidget() string {
	target := m.Target
	warning := "Configuration will be backed up before firmware is replaced."
	if m.Operation == "install" {
		warning = "This will erase and replace all flash contents on the selected device."
		if target.Image.InstallHint != "" {
			warning += "\n" + target.Image.InstallHint
		}
	}
	return lipgloss.NewStyle().Foreground(ColorWarn).Render(fmt.Sprintf(
		"Confirm %s of %s on %s?  %s yes  %s no\n%s",
		m.Operation, target.Image.Name, target.Device.Port, StyleKey.Render("Y"), StyleKey.Render("N"), strings.TrimSpace(warning),
	))
}
