package views

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/micros/microsctl/internal/tui/widgets"
)

func ShellView(m widgets.State) tea.View {
	width := max(20, m.Width-2)
	clean := m.ShellLines()
	height := m.ShellViewportHeight()
	offset := max(0, min(m.ShellScroll, max(0, len(clean)-height)))
	end := len(clean) - offset
	clean = clean[max(0, end-height):end]
	status := widgets.Clean(m.ShellStatus)
	if offset > 0 {
		status += fmt.Sprintf(" · scrollback: %d lines · → newer", offset)
	}
	input := widgets.Clean(m.ShellInput)
	prompt := widgets.Clean(m.ShellPrompt)
	if strings.Contains(prompt, "[password]") {
		input = strings.Repeat("*", len([]rune(input)))
	}
	available := max(1, width-ansi.StringWidth(prompt)-1)
	if ansi.StringWidth(input) > available {
		input = ansi.TruncateLeft(input, ansi.StringWidth(input)-available+1, "…")
	}
	entry := ""
	if m.ShellReady {
		entry = lipgloss.NewStyle().Bold(true).Render(prompt) + input + "▏"
	}
	output := strings.Join(clean, "\n")
	text := widgets.StyleTitle.Render("micrOS / Nodes / Shell") + "\n" + widgets.Clean(m.ShellAddress) + "\n\n" + output + "\n" + entry + "\n\n" + lipgloss.NewStyle().Foreground(widgets.ColorRelease).Render(status) + "\n" + widgets.StyleMuted.Render("enter send · ↑/↓ commands · ←/→ scroll · ctrl+u clear input · esc/ctrl+c disconnect and return")
	v := tea.NewView(text)
	v.AltScreen = true
	return v
}
