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
					info = fmt.Sprintf("%s (%d/%d)", m.Inventory.Devices[m.DeviceIndex].Port, m.DeviceIndex+1, len(m.Inventory.Devices))
				}
			}
			title += "  ·  " + StyleMuted.Render(info)
		}
		if item == "Discovery" {
			info := "scan and identify"
			if len(m.Inventory.Devices) > 0 {
				info = "not identified"
				if deviceInfo := m.Inventory.Devices[m.DeviceIndex].Info; deviceInfo != nil {
					info = deviceInfo.Chip
				}
			}
			if index == m.Cursor && m.UsbScanRequested && m.LoadingInventory {
				info = "scanning…"
			} else if m.ProbingUSB {
				info = "identifying…"
			}
			title += "  ·  " + StyleMuted.Render(info)
		}
		b.WriteString(marker + title + "\n")
		if item == "Discovery" && index == m.Cursor && !m.LoadingInventory && !m.ProbingUSB {
			if details := m.discoveryDetails(width); details != "" {
				b.WriteString(details + "\n")
			}
		}
	}
	return StylePanel.Width(width).Render(strings.TrimSuffix(b.String(), "\n"))
}

func (m State) discoveryDetails(width int) string {
	if m.DeviceIndex < 0 || m.DeviceIndex >= len(m.Inventory.Devices) {
		return ""
	}
	device := m.Inventory.Devices[m.DeviceIndex]
	if device.Info == nil {
		return ""
	}
	info := device.Info
	lines := []string{
		fmt.Sprintf("%-9s %s", "Port", device.Port),
		fmt.Sprintf("%-9s %s", "Revision", valueOrDash(info.Revision)),
		fmt.Sprintf("%-9s %s", "Flash", flashDescription(info.FlashSize, info.FlashID)),
		fmt.Sprintf("%-9s %s", "MAC", valueOrDash(info.MAC)),
		fmt.Sprintf("%-9s %s", "Features", valueOrDash(strings.Join(info.Features, ", "))),
	}
	for _, warning := range info.Warnings {
		lines = append(lines, fmt.Sprintf("%-9s %s", "Warning", warning))
	}
	return StyleMuted.Width(max(1, width-StylePanel.GetHorizontalFrameSize())).PaddingLeft(4).Render(strings.Join(lines, "\n"))
}

func valueOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	return value
}

func flashDescription(size, id string) string {
	if size == "" {
		return valueOrDash(id)
	}
	if id == "" {
		return size
	}
	return size + " · JEDEC " + id
}
