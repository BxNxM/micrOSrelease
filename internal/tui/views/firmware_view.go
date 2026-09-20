package views

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"github.com/micros/microsctl/internal/tui/widgets"
	"github.com/micros/microsctl/internal/usb"
	"strings"
)

func FirmwareView(m widgets.State) tea.View {
	width := max(20, min(100, m.Width-4))
	if m.Width == 0 {
		width = 88
	}
	selector := m.BoardSelectorWidget(width)
	indices := usb.BoardImages(m.Inventory.Images, m.BoardType)
	var images []usb.Image
	focused, selected := 0, -1
	for i, index := range indices {
		images = append(images, m.Inventory.Images[index])
		if index == m.FirmwareIndex {
			focused = i
		}
		if index == m.ImageIndex {
			selected = i
		}
	}
	m.Inventory.Images = images
	m.FirmwareIndex, m.ImageIndex = focused, selected
	var b strings.Builder
	b.WriteString(widgets.StyleTitle.Render("micrOS / 🔧 USB Tools / Firmware") + "\n")
	b.WriteString(widgets.StyleMuted.Render("Bundled frameworks · select an image for USB operations") + "\n\n")
	b.WriteString(selector + "\n\n")
	count := max(1, (m.Height-10-lipgloss.Height(selector))/5)
	start := m.FirmwareIndex / count * count
	if len(m.Inventory.Images) == 0 {
		b.WriteString("No firmware images bundled. Add .bin or .uf2 files to\nstorage/frameworks and rebuild.\n")
	}
	for i := start; i < min(start+count, len(m.Inventory.Images)); i++ {
		image := m.Inventory.Images[i]
		border, marker := widgets.ColorBorder, "  "
		if i == m.FirmwareIndex {
			border, marker = widgets.ColorAccent, "› "
		}
		selected := ""
		if m.FirmwareSelected && i == m.ImageIndex {
			selected = "  ✓ selected"
		}
		version := image.Version
		if version == "" {
			version = "unknown"
		}
		mp := image.MicroPython
		if mp == "" {
			mp = "unknown"
		}
		lines := []string{
			marker + strings.ToUpper(image.Board) + selected,
			image.Name,
			fmt.Sprintf("micrOS %s · MicroPython %s · %.2f MiB", version, mp, float64(image.Size)/(1024*1024)),
		}
		for j := range lines {
			lines[j] = widgets.Clean(lines[j])
		}
		micros := widgets.Clean("micrOS " + version)
		lines[2] = strings.Replace(lines[2], micros, lipgloss.NewStyle().Foreground(widgets.ColorAccent).Render(micros), 1)
		board := widgets.Clean(strings.ToUpper(image.Board))
		lines[0] = strings.Replace(lines[0], board, lipgloss.NewStyle().Bold(true).Render(board), 1)
		if m.FirmwareSelected && i == m.ImageIndex {
			lines[0] = strings.Replace(lines[0], "✓ selected", lipgloss.NewStyle().Foreground(widgets.ColorOnline).Render("✓ selected"), 1)
		}
		for j := range lines {
			lines[j] = ansi.Truncate(lines[j], width-4, "…")
		}
		b.WriteString(widgets.CardWidget(width, 3, border, lines...) + "\n")
	}
	b.WriteString(fmt.Sprintf("\n%d images · page %d/%d\n", len(m.Inventory.Images), start/count+1, max(1, (len(m.Inventory.Images)+count-1)/count)))
	b.WriteString(widgets.StyleMuted.Render("←/→ board · ↑/↓ browse · enter select · esc cancel · q quit"))
	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}
