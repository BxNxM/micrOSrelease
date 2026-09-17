package widgets

import (
	"fmt"
	"strings"
)

func (m State) ActionsMenuWidget(width int) string {
	var b strings.Builder
	board := "-"
	if m.FirmwareSelected && m.ImageIndex >= 0 && m.ImageIndex < len(m.Inventory.Images) {
		if selected := Clean(m.Inventory.Images[m.ImageIndex].Board); selected != "" {
			board = selected
		}
	}
	b.WriteString(StyleTitle.Render("USB ["+board+"]") + "\n\n")
	for index, item := range m.Actions {
		marker := "  "
		title := item
		if index == m.Cursor {
			marker = "› "
			title = StyleSelected.Render(title)
		}
		if item == "USB Scan" {
			info := "not scanned"
			if m.UsbScanRequested && m.LoadingInventory {
				info = "scanning…"
			} else if m.UsbScanned {
				info = "no devices found"
				if len(m.Inventory.Devices) > 0 {
					info = fmt.Sprintf("%s (%d/%d, demo)", m.Inventory.Devices[m.DeviceIndex].Port, m.DeviceIndex+1, len(m.Inventory.Devices))
				}
			}
			title += "  ·  " + StyleMuted.Render(info)
		}
		b.WriteString(marker + title + "\n")
	}
	return StylePanel.Width(width).Render(strings.TrimSuffix(b.String(), "\n"))
}
