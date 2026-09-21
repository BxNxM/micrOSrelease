package widgets

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// ShellCommand identifies a local command echo by byte offsets in the transcript.
// Server output is never interpreted as a command based on its text.
type ShellCommand struct {
	Start, PromptEnd, End int
}

// ShellLines supplies independently styled wrapped lines to rendering and scrolling.
func (s State) ShellLines() []string {
	width := max(20, s.Width-2)
	var lines []string
	offset, commandIndex := 0, 0
	for _, raw := range strings.Split(s.ShellOutput, "\n") {
		for commandIndex < len(s.ShellCommands) && s.ShellCommands[commandIndex].End <= offset {
			commandIndex++
		}
		command, promptLength := false, 0
		if commandIndex < len(s.ShellCommands) {
			span := s.ShellCommands[commandIndex]
			if span.Start <= offset && span.End > offset {
				command = true
				promptLength = len(Clean(raw[:max(0, min(len(raw), span.PromptEnd-offset))]))
			}
		}
		consumed := 0
		for _, line := range strings.Split(ansi.Hardwrap(Clean(raw), width, true), "\n") {
			styled := ""
			if command {
				end := max(0, min(len(line), promptLength-consumed))
				styled = lipgloss.NewStyle().Bold(true).Render(line[:end]) + line[end:]
			} else {
				styled = lipgloss.NewStyle().Foreground(lipgloss.Color("#AAB4BF")).Render(line)
			}
			lines = append(lines, styled)
			consumed += len(line)
		}
		offset += len(raw) + 1
	}
	return lines
}

func (s State) ShellViewportHeight() int { return max(1, s.Height-8) }
