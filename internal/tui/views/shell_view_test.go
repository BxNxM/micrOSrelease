package views

import (
	"strings"
	"testing"

	"github.com/micros/microsctl/internal/tui/widgets"
)

func TestShellViewShowsRecentOutputAndStripsControls(t *testing.T) {
	state := widgets.State{ShellReady: true, Width: 80, Height: 12, ShellAddress: "node", ShellPrompt: "node $ ", ShellInput: "hello", ShellOutput: "old\n1\n2\n3\n\x1b[31mrecent\x1b[0m", ShellStatus: "Connected"}
	text := ShellView(state).Content
	if strings.Contains(text, "old") || !strings.Contains(text, "recent") || !strings.Contains(text, "hello") || strings.Contains(text, "\x1b[31m") {
		t.Fatalf("unexpected terminal: %q", text)
	}
}

func TestShellPasswordIsMaskedAndPromptBold(t *testing.T) {
	text := ShellView(widgets.State{ShellReady: true, Width: 80, Height: 24, ShellPrompt: "[password] node $ ", ShellInput: "secret"}).Content
	if strings.Contains(text, "secret") || !strings.Contains(text, "******") || !strings.Contains(text, "\x1b[1m[password] node $ ") {
		t.Fatalf("unexpected password prompt: %q", text)
	}
}

func TestShellViewScrollbackWindow(t *testing.T) {
	state := widgets.State{Width: 80, Height: 12, ShellReady: true, ShellPrompt: "node $ ", ShellInput: "draft", ShellOutput: "oldest\none\ntwo\nthree\nfour\nlatest", ShellScroll: 2}
	text := ShellView(state).Content
	if !strings.Contains(text, "oldest") || strings.Contains(text, "latest") || !strings.Contains(text, "draft") || !strings.Contains(text, "scrollback: 2 lines") {
		t.Fatalf("unexpected scrollback: %q", text)
	}
	state.ShellScroll = 0
	text = ShellView(state).Content
	if strings.Contains(text, "oldest") || !strings.Contains(text, "latest") || strings.Contains(text, "scrollback:") {
		t.Fatalf("unexpected bottom: %q", text)
	}
	state.ShellScroll = 999
	if !strings.Contains(ShellView(state).Content, "oldest") {
		t.Fatal("out-of-range offset not clamped")
	}
}

func TestShellScrollbackRetainsCommandStyling(t *testing.T) {
	command := "node $ hello"
	state := widgets.State{Width: 80, Height: 12, ShellOutput: command + "\nreply\na\nb\nc\nd", ShellCommands: []widgets.ShellCommand{{Start: 0, PromptEnd: len("node $ "), End: len(command)}}, ShellScroll: 2}
	text := ShellView(state).Content
	if !strings.Contains(text, "\x1b[1mnode $ ") || !strings.Contains(text, "\x1b[38;2;170;180;191mreply") {
		t.Fatalf("scrollback styling lost: %q", text)
	}
	state.ShellScroll = 0
	if strings.Contains(ShellView(state).Content, "node $ hello") {
		t.Fatal("scroll did not move")
	}
	state.ShellScroll = 2
	if ShellView(state).Content != text {
		t.Fatal("scrolling changed styling")
	}
}
