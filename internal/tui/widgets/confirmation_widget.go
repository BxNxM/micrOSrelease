package widgets

import (
	"charm.land/lipgloss/v2"
	"fmt"
)

func (m State) ConfirmationWidget() string {
	target := m.Target
	warning := "This is a dummy operation; no hardware will be changed."
	if m.Operation == "install" {
		warning = "Production install will erase the selected device. This dummy will not."
	}
	return lipgloss.NewStyle().Foreground(ColorWarn).Render(fmt.Sprintf(
		"Confirm %s of %s on %s?  %s yes  %s no\n%s",
		m.Operation, target.Image.Name, target.Device.Port, StyleKey.Render("Y"), StyleKey.Render("N"), warning,
	))
}
