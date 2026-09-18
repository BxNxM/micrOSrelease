package views

import (
	"fmt"
	"github.com/micros/microsctl/internal/tui/widgets"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// nodeDetailsView displays the selected node's last scan results.
func NodeDetailsView(m widgets.State) tea.View {
	node := m.Nodes[m.NodeIndex-1]
	width, _, _ := m.NodeGrid()
	state, nameColor := "OFFLINE", widgets.ColorWarn
	latency := "n/a"
	if node.Online {
		state, nameColor = "ONLINE", widgets.ColorOnline
		latency = fmt.Sprintf("%.3fs", node.Latency.Seconds())
	}
	var b strings.Builder
	b.WriteString(widgets.StyleTitle.Render("micrOS / Nodes / Device details") + "\n\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(nameColor).Render(widgets.Clean(node.Name)) + "\n\n")
	fields := [][2]string{
		{"UID", node.UID}, {"Address", node.Address}, {"Status", state},
		{"Version", node.Version}, {"Mode", node.Mode}, {"Comm time", latency},
	}
	if !node.CheckedAt.IsZero() {
		fields = append(fields, [2]string{"Checked", node.CheckedAt.Local().Format("2006-01-02 15:04:05")})
	}
	if node.Cached {
		fields = append(fields, [2]string{"Source", "Saved observation (not checked this session)"})
	}
	for _, key := range widgets.FeatureKeys {
		fields = append(fields, [2]string{strings.ToUpper(key), node.Features[key]})
	}
	for _, field := range fields {
		value := widgets.Clean(field[1])
		if value == "" {
			value = "n/a"
		}
		valueStyle := lipgloss.NewStyle().Bold(false)
		switch field[0] {
		case "Status":
			valueStyle = valueStyle.Foreground(nameColor)
		case "Mode":
			value = widgets.ModeWidget(value)
		case "WEBUI", "ESPNOW", "CRON", "TIMIRQ":
			value = widgets.FeatureValueWidget(value)
		}
		value = valueStyle.Render(value)
		fmt.Fprintf(&b, "%-10s %s\n", field[0], value)
	}
	link := "-"
	if url := node.WebUIURL(); url != "" {
		link = url
		if m.DetailAction == 0 {
			link = widgets.StyleSelected.Underline(true).Render("› " + url)
		}
	}
	fmt.Fprintf(&b, "\n%-10s %s\n", "Web UI", link)
	remove := "  🗑 Remove device"
	if m.DetailAction == 1 {
		remove = widgets.StyleSelected.Render("› 🗑 Remove device")
	}
	b.WriteString("\n" + remove + "\n")
	if node.Error != "" {
		b.WriteString("\n" + widgets.Clean(node.Error) + "\n")
	}
	help := "↑/↓ select · enter activate · esc back to grid · q quit"
	if node.WebUIURL() != "" {
		help = "enter/o open Web UI · " + help
	}
	b.WriteString("\n" + widgets.StyleMuted.Render(widgets.Clean(m.Status)))
	b.WriteString("\n" + widgets.StyleMuted.Render(help))
	v := tea.NewView(lipgloss.NewStyle().Width(width).Render(b.String()))
	v.AltScreen = true
	return v
}
