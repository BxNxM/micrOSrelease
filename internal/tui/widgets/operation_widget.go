package widgets

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/micros/microsctl/internal/usb"
)

func (m State) OperationWidget(width int) string {
	var b strings.Builder
	b.WriteString(StyleTitle.Render(strings.ToUpper(string(m.Operation))))
	b.WriteString("  ")
	if m.Running {
		frames := []string{"◐", "◓", "◑", "◒"}
		b.WriteString(fmt.Sprintf("%s %d%%", frames[m.SpinnerFrame%len(frames)], m.Progress))
	} else if m.Result != nil {
		b.WriteString(StyleSelected.Render("✓ complete"))
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(ColorError).Render("✗ stopped"))
	}
	b.WriteString("\n\n")
	barWidth := max(10, width-12)
	filled := min(100, max(0, m.Progress)) * barWidth / 100
	b.WriteString(lipgloss.NewStyle().Foreground(ColorAccent).Render(strings.Repeat("━", filled)))
	b.WriteString(lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("━", barWidth-filled)))
	b.WriteString("\n\n")
	canReconnect := false
	for _, stage := range m.Stages {
		marker, color := "○", ColorMuted
		switch stage.State {
		case usb.StageRunning:
			frames := []string{"◐", "◓", "◑", "◒"}
			marker, color = frames[m.SpinnerFrame%len(frames)], ColorWarn
		case usb.StageDone:
			marker, color = "✓", ColorOnline
		case usb.StageSkipped:
			marker, color = "–", ColorMuted
		case usb.StageFailed:
			marker, color = "✗", ColorError
		}
		b.WriteString(lipgloss.NewStyle().Foreground(color).Render(marker+" "+Clean(stage.Name)+" · "+string(stage.State)) + "\n")
		if stage.State == usb.StageRunning && stage.Detail != "" {
			detail := ansi.Truncate("  "+Clean(stage.Detail), max(1, width-StylePanel.GetHorizontalFrameSize()), "…")
			b.WriteString(StyleMuted.Render(detail) + "\n")
			canReconnect = canReconnect || stage.CanReconnect
		}
	}
	if m.OperationError != "" {
		errorText := Clean(strings.ReplaceAll(m.OperationError, "\n", "; "))
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(ColorError).
			Width(max(1, width-StylePanel.GetHorizontalFrameSize())).Render("Error: "+errorText) + "\n")
	}
	hint := "Do not disconnect the USB device during this operation."
	if canReconnect {
		hint = "USB reconnect · unplug/replug is safe · use the same USB socket · Ctrl+C cancels"
	} else if m.OperationError != "" {
		hint = "Review the error above before retrying."
	}
	b.WriteString("\n" + StyleMuted.Render(hint))
	return StylePanel.Width(width).Render(strings.TrimSuffix(b.String(), "\n"))
}
