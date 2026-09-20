package widgets

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

func (m State) ReleaseTargetWidget(width int) string {
	device := "No USB device"
	if len(m.Inventory.Devices) > 0 {
		device = m.Inventory.Devices[m.DeviceIndex].Port
	}
	image, board := "Not selected — press f", "—"
	micropython, micros := "n/a", "n/a"
	if m.FirmwareSelected && len(m.Inventory.Images) > 0 {
		selected := m.Inventory.Images[m.ImageIndex]
		image, board = selected.Name, selected.Board
		if selected.MicroPython != "" {
			micropython = selected.MicroPython
		}
		if selected.Version != "" {
			micros = selected.Version
		}
	}
	hintStyle := StyleMuted
	if !m.FirmwareSelected {
		hintStyle = hintStyle.Foreground(ColorWarn)
	}
	content := StyleTitle.Render("CHOOSE FIRMWARE") + "  " + hintStyle.Render("[f] choose firmware") + "\n" +
		fmt.Sprintf("  %-12s %s\n", "Board", board) +
		fmt.Sprintf("  %-12s %s\n", "MicroPython", micropython) +
		fmt.Sprintf("  %-12s %s\n", "micrOS", lipgloss.NewStyle().Bold(true).Render(micros)) +
		fmt.Sprintf("  %-12s %s\n", "USB", StyleSelected.Render(device)) +
		fmt.Sprintf("  %-12s %s", "Image", image)
	return StylePanel.PaddingTop(0).Width(width).Render(m.BoardSelectorWidget(width-StylePanel.GetHorizontalFrameSize()) + "\n\n" + content)
}
