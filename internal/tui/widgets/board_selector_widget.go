package widgets

import (
	"charm.land/lipgloss/v2"
	"github.com/micros/microsctl/internal/usb"
	"strings"
)

// BoardSelectorWidget shares bordered board tabs across USB Tools and firmware selection.
func (m State) BoardSelectorWidget(width int) string {
	var rows, tabs []string
	rowWidth := 0
	hint := StyleMuted.Render("←/→ board")
	boards := usb.Boards(m.Inventory.Images)
	for i, board := range boards {
		style := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).Foreground(ColorText).Padding(0, 2)
		if board == m.BoardType {
			style = style.BorderForeground(ColorAccent)
		}
		tab := style.Render(Clean(board))
		tabWidth := lipgloss.Width(tab)
		requiredWidth := tabWidth
		if i == len(boards)-1 {
			requiredWidth += 2 + lipgloss.Width(hint)
		}
		if len(tabs) > 0 && rowWidth+1+requiredWidth > width {
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, tabs...))
			tabs, rowWidth = nil, 0
		}
		if len(tabs) > 0 {
			tabs = append(tabs, " ")
			rowWidth++
		}
		tabs = append(tabs, tab)
		rowWidth += tabWidth
	}
	if len(tabs) > 0 {
		if rowWidth+2+lipgloss.Width(hint) <= width {
			tabs = append(tabs, "  ", hint)
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Center, tabs...))
			return strings.Join(rows, "\n")
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, tabs...))
	}
	rows = append(rows, hint)
	return strings.Join(rows, "\n")
}
