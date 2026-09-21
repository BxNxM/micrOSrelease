package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/micros/microsctl/internal/network"
	"github.com/micros/microsctl/internal/tui/widgets"
)

type shellState struct {
	commands      []widgets.ShellCommand
	scroll        int
	visible       bool
	generation    int
	node          network.Device
	session       network.Shell
	ctx           context.Context
	cancel        context.CancelFunc
	busy          bool
	input         string
	output        string
	status        string
	prompt        string
	stream        string
	history       []string
	historyOffset int
	draft         string
}

type shellOpenedMsg struct {
	generation int
	session    network.Shell
	err        error
}
type shellReplyMsg struct {
	partial    bool
	exit       bool
	queue      <-chan shellReplyMsg
	prompt     string
	generation int
	output     string
	err        error
}

func (m model) openShell(node network.Device) (tea.Model, tea.Cmd) {
	ctx, cancel := context.WithCancel(context.Background())
	m.shell = shellState{visible: true, generation: m.shell.generation + 1, node: node, ctx: ctx, cancel: cancel, busy: true, status: "Connecting…"}
	generation := m.shell.generation
	connector, ok := m.network.(network.ShellConnector)
	return m, func() tea.Msg {
		if !ok {
			return shellOpenedMsg{generation: generation, err: fmt.Errorf("shell unavailable")}
		}
		session, err := connector.OpenShell(ctx, node)
		if ctx.Err() != nil && session != nil {
			session.Close()
			session = nil
			err = ctx.Err()
		}
		return shellOpenedMsg{generation: generation, session: session, err: err}
	}
}

func (m model) handleShellOpened(msg shellOpenedMsg) (tea.Model, tea.Cmd) {
	if !m.shell.visible || msg.generation != m.shell.generation {
		if msg.session != nil {
			return m, func() tea.Msg { msg.session.Close(); return nil }
		}
		return m, nil
	}
	m.shell.busy = false
	m.shell.session = msg.session
	m.shell.status = "Connected"
	if msg.session != nil {
		m.shell.prompt = msg.session.Prompt()
		m.shell.output = msg.session.Greeting()
	}
	if msg.err != nil {
		m.shell.status = "Connection failed: " + msg.err.Error()
	}
	return m, nil
}

func (m model) handleShellReply(msg shellReplyMsg) (tea.Model, tea.Cmd) {
	if !m.shell.visible || msg.generation != m.shell.generation {
		return m, nil
	}
	previousLines := 0
	if m.shell.scroll > 0 {
		previousLines = len(m.renderState().ShellLines())
	}
	if msg.partial {
		m.shell.stream = shellTail(msg.output)
		m.preserveShellScroll(previousLines)
		return m, nextShellReply(msg.queue)
	}
	m.shell.busy = false
	m.shell.stream = ""
	m.shell.prompt = msg.prompt
	m.shell.output += msg.output
	if m.shell.output != "" && !strings.HasSuffix(m.shell.output, "\n") {
		m.shell.output += "\n"
	}
	// Keep a bounded terminal transcript.
	if len(m.shell.output) > 64*1024 {
		removed := len(m.shell.output) - 64*1024
		m.shell.output = m.shell.output[removed:]
		var commands []widgets.ShellCommand
		for _, command := range m.shell.commands {
			if command.End <= removed {
				continue
			}
			command.Start = max(0, command.Start-removed)
			command.PromptEnd = max(0, command.PromptEnd-removed)
			command.End -= removed
			commands = append(commands, command)
		}
		m.shell.commands = commands
	}
	m.preserveShellScroll(previousLines)
	m.shell.status = "Connected"
	if msg.err != nil {
		m.shell.status = "Disconnected: " + msg.err.Error()
		if msg.exit && errors.Is(msg.err, network.ErrSessionClosed) {
			m.shell.visible = false
			m.status = "Shell session closed"
		}
		m.shell.input = ""
		if m.shell.cancel != nil {
			m.shell.cancel()
		}
		session := m.shell.session
		m.shell.session = nil
		return m, func() tea.Msg {
			if session != nil {
				session.Close()
			}
			return nil
		}
	}
	return m, nil
}

func (m model) submitShell() (tea.Model, tea.Cmd) {
	if m.shell.busy || m.shell.session == nil || strings.TrimSpace(m.shell.input) == "" {
		return m, nil
	}
	m.shell.scroll = 0
	command := m.shell.input
	m.shell.input = ""
	echo := command
	if strings.Contains(m.shell.prompt, "[password]") {
		echo = "********"
	} else if len(m.shell.history) == 0 || m.shell.history[len(m.shell.history)-1] != command {
		m.shell.history = append(m.shell.history, command)
		if len(m.shell.history) > 100 {
			m.shell.history = append([]string(nil), m.shell.history[len(m.shell.history)-100:]...)
		}
	}
	m.shell.historyOffset = 0
	m.shell.draft = ""
	start := len(m.shell.output)
	m.shell.commands = append(m.shell.commands, widgets.ShellCommand{Start: start, PromptEnd: start + len(m.shell.prompt), End: start + len(m.shell.prompt) + len(echo)})
	m.shell.output += m.shell.prompt + echo + "\n"
	m.shell.busy = true
	m.shell.status = "Waiting for reply…"
	session, ctx, generation := m.shell.session, m.shell.ctx, m.shell.generation
	queue := make(chan shellReplyMsg, 8)
	exit := strings.TrimSpace(command) == "exit" && !strings.Contains(m.shell.prompt, "[password]")
	return m, func() tea.Msg {
		go func() {
			defer close(queue)
			emit := func(output string) {
				select {
				case queue <- shellReplyMsg{generation: generation, output: shellTail(output), partial: true}:
				case <-ctx.Done():
				}
			}
			output, err := session.StreamCommand(ctx, command, emit)
			select {
			case queue <- shellReplyMsg{generation: generation, output: shellTail(output), prompt: session.Prompt(), err: err, exit: exit}:
			case <-ctx.Done():
			}
		}()
		return nextShellReply(queue)()
	}
}

func shellTail(output string) string {
	if len(output) > 64*1024 {
		return output[len(output)-64*1024:]
	}
	return output
}

func nextShellReply(queue <-chan shellReplyMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-queue
		if !ok {
			return nil
		}
		msg.queue = queue
		return msg
	}
}
