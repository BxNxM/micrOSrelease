package views

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"fmt"
	"github.com/charmbracelet/x/ansi"
	"github.com/micros/microsctl/internal/tui/widgets"
	"strings"
)

func NodesView(m widgets.State) tea.View {
	width, columns, _ := m.NodeGrid()
	cardWidth := (width - (columns - 1)) / columns
	var b strings.Builder
	b.WriteString(widgets.StyleTitle.Render("micrOS / Nodes 📡") + "\n")
	if m.AppUpdateLine != "" {
		b.WriteString(ansi.Truncate(widgets.StyleMuted.Foreground(widgets.ColorRelease).Render(widgets.Clean(m.AppUpdateLine)), width, "…") + "\n")
	}
	b.WriteString(widgets.StyleMuted.Render("TCP 9008 · auto-refresh 5 min") + "\n\n")
	updated := "n/a"
	if !m.LastUpdated.IsZero() {
		updated = m.LastUpdated.Local().Format("2006-01-02 15:04:05")
	}
	activity := widgets.StyleMuted.Render("● Last updated: " + updated)
	if m.Discovering {
		activity = widgets.ActivityWidget("Network scan & status refresh")
	}
	b.WriteString(ansi.Truncate(activity, width, "…") + "\n")
	if len(m.Nodes) == 0 {
		if !m.Discovering {
			b.WriteString("No nodes found. Press r to scan.")
		}
		b.WriteString("\n")
	}
	count := m.CardsPerPage()
	pages := max(1, (len(m.Nodes)+1+count-1)/count)
	page := min(m.NodeIndex/count, pages-1)
	start := page * count
	var cards []string
	for index := start; index < min(start+count, len(m.Nodes)+1); index++ {
		if index == 0 {
			border := lipgloss.Color("#8060A8")
			if m.NodeIndex == 0 {
				border = lipgloss.Color("#D6A2FF")
			}
			cards = append(cards, widgets.CardWidget(cardWidth, 4, border, widgets.StyleTitle.Foreground(border).Render("🔧 USB Tools"), "USB install & update", "Release tools", "Enter to open"))
			continue
		}
		node := m.Nodes[index-1]
		color, nameColor := widgets.NodeColors(node, index == m.NodeIndex)
		latency := "n/a"
		if node.Online {
			latency = fmt.Sprintf("%.3fs", node.Latency.Seconds())
		}
		if node.Cached {
			latency += " · saved"
		}
		lines := []string{
			lipgloss.NewStyle().Bold(true).Foreground(nameColor).Render(widgets.NodeTitle(node)) + " · " + widgets.Clean(node.Version),
			fmt.Sprintf("%s · %s", widgets.ModeWidget(node.Mode), latency),
		}
		switch node.SpecialEndpoint() {
		case "Localhost":
			lines[1] = "127.0.0.1 · " + lines[1]
		case "AP mode":
			lines[1] = "192.168.4.1 · " + lines[1]
		}
		var features []string
		for _, key := range widgets.FeatureKeys {
			features = append(features, strings.ToUpper(key)+": "+widgets.FeatureValueWidget(node.Features[key]))
		}
		lines = append(lines, strings.Join(features[:3], " · "), strings.Join(features[3:], " · "))
		cards = append(cards, widgets.CardWidget(cardWidth, 4, color, lines...))
	}
	for i := 0; i < len(cards); i += columns {
		var row []string
		for j := i; j < min(i+columns, len(cards)); j++ {
			if j > i {
				row = append(row, " ")
			}
			row = append(row, cards[j])
		}
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, row...) + "\n")
	}
	if m.NodeIndex > 0 && m.NodeIndex <= len(m.Nodes) {
		node := m.Nodes[m.NodeIndex-1]
		if node.Error != "" {
			b.WriteString("\n" + ansi.Truncate(widgets.Clean(node.Error), width, "…"))
		}
	}
	b.WriteString("\n" + ansi.Truncate(widgets.Clean(m.Status), width, "…"))
	b.WriteString(fmt.Sprintf("\n↑↓←→ select · enter open · page %d/%d · r scan · q quit", page+1, pages))
	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}
