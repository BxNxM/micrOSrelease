package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/micros/microsctl/internal/network"
)

type fakeShell struct {
	commands []string
	closed   bool
}

func (s *fakeShell) Command(_ context.Context, command string) (string, error) {
	s.commands = append(s.commands, command)
	return "reply", nil
}
func (s *fakeShell) Greeting() string { return "" }
func (s *fakeShell) StreamCommand(ctx context.Context, command string, emit func(string)) (string, error) {
	output, err := s.Command(ctx, command)
	emit("rep")
	emit(output)
	return output, err
}
func (s *fakeShell) Prompt() string { return "node $ " }
func (s *fakeShell) Close() error   { s.closed = true; return nil }

type shellNetwork struct{ session *fakeShell }

func (s shellNetwork) Discover(context.Context, func(network.Device)) error { return nil }
func (s shellNetwork) OpenShell(context.Context, network.Device) (network.Shell, error) {
	return s.session, nil
}

func TestShellLifecycle(t *testing.T) {
	session := &fakeShell{}
	m := New(nil, shellNetwork{session})
	m.nodes = []network.Device{{Name: "node", Address: "127.0.0.1:9008", UID: "id", Online: true}}
	m.nodeIndex, m.showNodeDetails, m.detailAction = 1, true, 0
	next, _ := m.handleKey("down")
	m = next.(model)
	next, cmd := m.handleKey("enter")
	m = next.(model)
	if !m.shell.visible || !m.shell.busy {
		t.Fatal("shell did not open")
	}
	next, _ = m.Update(cmd())
	m = next.(model)
	for _, key := range []string{"q", "space", "j"} {
		next, _ = m.handleKey(key)
		m = next.(model)
	}
	if m.shell.input != "q j" {
		t.Fatalf("typing intercepted: %q", m.shell.input)
	}
	next, cmd = m.handleKey("enter")
	m = next.(model)
	if _, duplicate := m.handleKey("enter"); duplicate != nil {
		t.Fatal("concurrent command")
	}
	for cmd != nil {
		next, cmd = m.Update(cmd())
		m = next.(model)
	}
	if len(session.commands) != 1 || session.commands[0] != "q j" || m.shell.output != "node $ q j\nreply\n" {
		t.Fatalf("unexpected exchange: %+v", m.shell)
	}
	next, _ = m.Update(tea.PasteMsg{Content: "hello\nworld"})
	m = next.(model)
	if m.shell.input != "helloworld" {
		t.Fatal("paste allowed control characters")
	}
	next, cmd = m.handleKey("esc")
	m = next.(model)
	cmd()
	if m.shell.visible || !session.closed || m.shell.ctx.Err() == nil || !m.showNodeDetails {
		t.Fatal("session not cleaned up")
	}
}

func TestShellIgnoresLateMessagesAndClosesLateConnection(t *testing.T) {
	m := model{shell: shellState{generation: 2, visible: true, status: "new"}}
	session := &fakeShell{}
	next, cmd := m.Update(shellOpenedMsg{generation: 1, session: session})
	m = next.(model)
	cmd()
	if !session.closed || m.shell.session != nil {
		t.Fatal("stale connection retained")
	}
	next, _ = m.Update(shellReplyMsg{generation: 1, output: "old", err: errors.New("old")})
	m = next.(model)
	if m.shell.output != "" || m.shell.status != "new" {
		t.Fatal("stale reply applied")
	}
}

func TestShellConnectionAndCommandErrors(t *testing.T) {
	m := model{shell: shellState{visible: true, generation: 1, busy: true}}
	next, _ := m.Update(shellOpenedMsg{generation: 1, err: errors.New("auth failed")})
	m = next.(model)
	if m.shell.busy || m.shell.status != "Connection failed: auth failed" {
		t.Fatal(m.shell.status)
	}
	session := &fakeShell{}
	m.shell.session = session
	next, cmd := m.Update(shellReplyMsg{generation: 1, err: errors.New("timeout")})
	m = next.(model)
	cmd()
	if !session.closed || m.shell.session != nil || m.shell.status != "Disconnected: timeout" {
		t.Fatal(m.shell.status)
	}
}

func TestShellPasswordEchoAndExit(t *testing.T) {
	session := &fakeShell{}
	m := model{shell: shellState{visible: true, generation: 1, session: session, ctx: context.Background(), prompt: "[password] node $ ", input: "secret"}}
	next, _ := m.submitShell()
	m = next.(model)
	if strings.Contains(m.shell.output, "secret") || !strings.Contains(m.shell.output, "********") {
		t.Fatal("password leaked")
	}
	next, _ = m.Update(shellReplyMsg{generation: 1, prompt: "[configure] node $ "})
	m = next.(model)
	if m.shell.prompt != "[configure] node $ " {
		t.Fatal("prompt not updated")
	}
	next, cmd := m.Update(shellReplyMsg{generation: 1, err: network.ErrSessionClosed, exit: true})
	m = next.(model)
	cmd()
	if m.shell.visible || !session.closed {
		t.Fatal("exit did not close terminal")
	}
}

func TestShellPartialOutputDoesNotMakePromptReady(t *testing.T) {
	session := &fakeShell{}
	m := model{shell: shellState{visible: true, generation: 1, session: session, busy: true, output: "node $ help\n", prompt: "node $ "}}
	next, _ := m.Update(shellReplyMsg{generation: 1, output: "first\n", partial: true})
	m = next.(model)
	if !m.shell.busy || m.renderState().ShellReady || m.renderState().ShellOutput != "node $ help\nfirst\n" {
		t.Fatal("partial output not visible or ready too early")
	}
	next, _ = m.Update(shellReplyMsg{generation: 1, output: "first\nsecond\n", prompt: "[configure] node $ "})
	m = next.(model)
	if !m.renderState().ShellReady || m.renderState().ShellOutput != "node $ help\nfirst\nsecond\n" || m.shell.prompt != "[configure] node $ " {
		t.Fatal("completion duplicated output or lost prompt")
	}
}
