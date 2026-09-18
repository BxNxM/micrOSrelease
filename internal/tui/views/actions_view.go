package views

import (
	"github.com/micros/microsctl/internal/tui/widgets"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func ActionsView(m widgets.State) tea.View {
	terminalWidth := m.Width
	if terminalWidth <= 0 {
		terminalWidth = 96
	}
	contentWidth := max(48, min(terminalWidth-4, 104))
	panelWidth := contentWidth - 6

	var page strings.Builder
	page.WriteString(widgets.StyleLogo.Render("micrOS"))
	page.WriteString("  ")
	page.WriteString(widgets.StyleTitle.Render("/ USB Tools"))
	page.WriteString("\n")
	page.WriteString(widgets.StyleMuted.Render("Native ESP install · state-preserving USB update"))
	page.WriteString("\n\n")
	page.WriteString(m.ReleaseTargetWidget(panelWidth))
	page.WriteString("\n")
	page.WriteString(m.ActionsMenuWidget(panelWidth))
	page.WriteString("\n")
	if m.Confirming {
		page.WriteString(m.ConfirmationWidget())
		page.WriteString("\n")
	}
	if m.Running || m.Result != nil || len(m.Stages) > 0 {
		page.WriteString(m.OperationWidget(panelWidth))
		page.WriteString("\n")
	}
	page.WriteString(m.StatusWidget())
	page.WriteString("\n\n")
	page.WriteString(widgets.StyleMuted.Render("↑/↓ action · ←/→ board · [/] USB · f firmware · enter select · esc nodes · q quit"))

	view := tea.NewView(lipgloss.NewStyle().Width(contentWidth).Render(page.String()))
	view.AltScreen = true
	return view
}
