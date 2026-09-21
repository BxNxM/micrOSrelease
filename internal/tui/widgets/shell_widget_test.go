package widgets

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestShellLinesKeepPromptCommandAndResponseStyles(t *testing.T) {
	prompt := "[configure] node $ "
	command := prompt + "version"
	state := State{Width: 80, ShellOutput: command + "\nnode $ server text\n", ShellCommands: []ShellCommand{{Start: 0, PromptEnd: len(prompt), End: len(command)}}}
	lines := state.ShellLines()
	if !strings.Contains(lines[0], "\x1b[1m"+prompt) || !strings.HasSuffix(lines[0], "version") || strings.Contains(lines[0], "38;2;170;180;191") {
		t.Fatalf("command lost style: %q", lines[0])
	}
	if !strings.Contains(lines[1], "38;2;170;180;191") || strings.Contains(lines[1], "\x1b[1m") {
		t.Fatalf("server text treated as command: %q", lines[1])
	}
}

func TestWrappedShellPromptLinesAreIndependentlyStyled(t *testing.T) {
	prompt := "[configure] TinyDevBoard $ "
	command := prompt + "version"
	state := State{Width: 22, ShellOutput: command, ShellCommands: []ShellCommand{{Start: 0, PromptEnd: len(prompt), End: len(command)}}}
	lines := state.ShellLines()
	if len(lines) != 2 {
		t.Fatalf("wrapped lines: %q", lines)
	}
	for _, line := range lines {
		if !strings.Contains(line, "\x1b[1m") {
			t.Fatalf("wrapped prompt lost bold: %q", line)
		}
	}
	if ansi.Strip(strings.Join(lines, "")) != command {
		t.Fatalf("text changed: %q", lines)
	}
}
