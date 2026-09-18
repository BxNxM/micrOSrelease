package widgets

import (
	"charm.land/lipgloss/v2"
	"fmt"
	"github.com/micros/microsctl/internal/usb"
	"strings"
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
			b.WriteString(StyleMuted.Render("  "+Clean(stage.Detail)) + "\n")
			canReconnect = canReconnect || stage.CanReconnect
		}
	}
	hint := "Do not disconnect the USB device during this operation."
	if canReconnect {
		hint = "Waiting for USB · unplug/replug is safe at this stage · resumes automatically · Ctrl+C cancels"
	}
	b.WriteString("\n" + StyleMuted.Render(hint))
	return StylePanel.Width(width).Render(strings.TrimSuffix(b.String(), "\n"))
}
