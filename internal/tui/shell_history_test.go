package tui

import (
	"context"
	"fmt"
	"testing"

	"github.com/micros/microsctl/internal/network"
)

func historyModel() model {
	return model{shell: shellState{visible: true, session: &fakeShell{}, ctx: context.Background(), prompt: "node $ "}}
}

func rememberCommand(m model, command string) model {
	m.shell.input = command
	next, _ := m.submitShell()
	m = next.(model)
	m.shell.busy = false // The command completion does not change history.
	return m
}

func TestShellHistoryNavigationAndDraft(t *testing.T) {
	m := historyModel()
	m = rememberCommand(m, "hello")
	m = rememberCommand(m, "version")
	m.shell.input = "unfinished"
	for _, step := range []struct{ key, want string }{
		{"up", "version"}, {"up", "hello"}, {"up", "hello"},
		{"down", "version"}, {"down", "unfinished"}, {"down", "unfinished"},
		{"up", "version"}, {"backspace", "versio"}, {"down", "unfinished"}, {"up", "version"},
	} {
		next, _ := m.handleKey(step.key)
		m = next.(model)
		if m.shell.input != step.want {
			t.Fatalf("%s: got %q want %q", step.key, m.shell.input, step.want)
		}
	}
	m = rememberCommand(m, "help")
	next, _ := m.handleKey("up")
	m = next.(model)
	if m.shell.input != "help" {
		t.Fatal("new command not newest")
	}
	next, _ = m.handleKey("down")
	m = next.(model)
	if m.shell.input != "" {
		t.Fatal("submitted draft retained")
	}
}

func TestShellHistoryExcludesPasswordsEmptyAndRepeatedCommands(t *testing.T) {
	m := historyModel()
	m = rememberCommand(m, "hello")
	m = rememberCommand(m, "hello")
	m = rememberCommand(m, "  ")
	m.shell.prompt = "[password] node $ "
	m = rememberCommand(m, "secret")
	m.shell.input = "password draft"
	next, _ := m.handleKey("up")
	m = next.(model)
	if m.shell.input != "password draft" {
		t.Fatal("history active during password prompt")
	}
	if len(m.shell.history) != 1 || m.shell.history[0] != "hello" {
		t.Fatalf("unexpected history: %q", m.shell.history)
	}
	m.shell.prompt = "node $ "
	m.shell.busy = true
	next, _ = m.handleKey("up")
	m = next.(model)
	if m.shell.input != "password draft" {
		t.Fatal("history active while busy")
	}
}

func TestShellHistoryBoundAndNewSession(t *testing.T) {
	m := historyModel()
	for i := range 105 {
		m = rememberCommand(m, fmt.Sprintf("command %d", i))
	}
	if len(m.shell.history) != 100 || m.shell.history[0] != "command 5" {
		t.Fatal("history limit")
	}
	m.network = shellNetwork{&fakeShell{}}
	next, _ := m.openShell(network.Device{Name: "node"})
	m = next.(model)
	defer m.shell.cancel()
	if len(m.shell.history) != 0 || m.shell.historyOffset != 0 || m.shell.draft != "" {
		t.Fatal("history leaked into new session")
	}
}

func TestShellEmptyHistoryKeepsInput(t *testing.T) {
	m := historyModel()
	m.shell.input = "draft"
	for _, key := range []string{"up", "down"} {
		next, _ := m.handleKey(key)
		m = next.(model)
		if m.shell.input != "draft" {
			t.Fatal("empty history changed input")
		}
	}
}
