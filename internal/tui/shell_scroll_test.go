package tui

import (
	"strings"
	"testing"
)

func TestShellScrollKeysBoundsAndCommandHistory(t *testing.T) {
	m := historyModel()
	m.height, m.width = 12, 80
	m.shell.output = "zero\none\ntwo\nthree\nfour\nfive\nsix\nseven"
	m.shell.input = "draft"
	m.shell.history = []string{"hello"}
	for _, step := range []struct {
		key    string
		offset int
	}{
		{"left", 2}, {"left", 4}, {"left", 4}, {"right", 2}, {"right", 0}, {"right", 0},
	} {
		next, _ := m.handleKey(step.key)
		m = next.(model)
		if m.shell.scroll != step.offset || m.shell.input != "draft" {
			t.Fatalf("%s: scroll=%d input=%q", step.key, m.shell.scroll, m.shell.input)
		}
	}
	next, _ := m.handleKey("up")
	m = next.(model)
	if m.shell.input != "hello" || m.shell.scroll != 0 {
		t.Fatal("command history changed")
	}
	m.shell.busy = true
	next, _ = m.handleKey("left")
	m = next.(model)
	if m.shell.scroll != 2 {
		t.Fatal("cannot scroll during stream")
	}
}

func TestShellScrollPreservesPositionDuringStreamAndFollowsBottom(t *testing.T) {
	for _, offset := range []int{0, 2} {
		m := historyModel()
		m.height, m.width = 12, 80
		m.shell.busy = true
		m.shell.scroll = offset
		m.shell.output = "0\n1\n2\n3\n4\n5\n"
		next, _ := m.Update(shellReplyMsg{generation: m.shell.generation, partial: true, output: "6\n7\n"})
		m = next.(model)
		want := 0
		if offset > 0 {
			want = 4
		}
		if m.shell.scroll != want {
			t.Fatalf("offset=%d want=%d", m.shell.scroll, want)
		}
		next, _ = m.Update(shellReplyMsg{generation: m.shell.generation, output: "6\n7\n", prompt: "node $ "})
		m = next.(model)
		if m.shell.scroll != want {
			t.Fatal("completion moved scroll")
		}
		m.shell.input = "help"
		next, _ = m.submitShell()
		m = next.(model)
		if m.shell.scroll != 0 {
			t.Fatal("new command did not return to bottom")
		}
	}
}

func TestShellScrollUsesWrappedLinesAndClampsAfterResize(t *testing.T) {
	m := historyModel()
	m.width, m.height = 22, 12
	m.shell.output = strings.Repeat("x", 160)
	next, _ := m.handleKey("left")
	m = next.(model)
	if m.shell.scroll != 2 {
		t.Fatal("wrapped output not scrollable")
	}
	m.height = 40
	next, _ = m.handleKey("right")
	m = next.(model)
	if m.shell.scroll != 0 {
		t.Fatal("resize did not clamp scroll")
	}
}

func TestShellTranscriptTrimKeepsCommandRanges(t *testing.T) {
	m := historyModel()
	m.shell.output = strings.Repeat("x", 64*1024) + "\n"
	m.shell.input = "hello"
	next, _ := m.submitShell()
	m = next.(model)
	next, _ = m.Update(shellReplyMsg{generation: m.shell.generation, output: "reply\n", prompt: "node $ "})
	m = next.(model)
	if len(m.shell.output) > 64*1024 || len(m.shell.commands) != 1 {
		t.Fatal("transcript trim lost command metadata")
	}
	span := m.shell.commands[0]
	if m.shell.output[span.Start:span.PromptEnd] != "node $ " || m.shell.output[span.PromptEnd:span.End] != "hello" {
		t.Fatal("trim did not rebase command range")
	}
	next, _ = m.Update(shellReplyMsg{generation: m.shell.generation, output: strings.Repeat("y", 64*1024)})
	m = next.(model)
	if len(m.shell.commands) != 0 {
		t.Fatal("discarded transcript retained command metadata")
	}
}
