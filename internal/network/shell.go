package network

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Shell is a sequential, persistent micrOS command session.
type Shell interface {
	Command(context.Context, string) (string, error)
	StreamCommand(context.Context, string, func(string)) (string, error)
	Greeting() string
	Close() error
	Prompt() string
}

type ShellConnector interface {
	OpenShell(context.Context, Device) (Shell, error)
}

func (s *Service) OpenShell(ctx context.Context, node Device) (Shell, error) {
	client, err := dial(ctx, node.Address, "", true)
	if err != nil {
		return nil, err
	}
	session := &nodeShell{Client: client, node: node}
	client.timeout = 20 * time.Second
	if !client.PasswordRequired() {
		if err := session.verify(ctx); err != nil {
			client.Close()
			return nil, err
		}
	}
	return session, nil
}

type nodeShell struct {
	*Client
	node Device
}

func (s *nodeShell) Greeting() string { return s.greeting }

func (s *nodeShell) Command(ctx context.Context, command string) (string, error) {
	output, err := s.StreamCommand(ctx, command, nil)
	return strings.TrimSpace(output), err
}

func (s *nodeShell) StreamCommand(ctx context.Context, command string, emit func(string)) (string, error) {
	authenticating := s.PasswordRequired()
	if !authenticating && strings.TrimSpace(command) == "exit" {
		s.deadline(ctx)
		stop := context.AfterFunc(ctx, func() { s.Close() })
		defer stop()
		_, err := fmt.Fprint(s.conn, command)
		s.Close()
		if err != nil {
			return "", err
		}
		return "Bye!", ErrSessionClosed
	}
	output, err := s.Client.StreamCommand(ctx, command, emit)
	if err == nil && authenticating {
		if s.PasswordRequired() {
			err = fmt.Errorf("authentication failed")
		} else {
			err = s.verify(ctx)
		}
	}
	if err != nil {
		s.Close()
	}
	return output, err
}

func (s *nodeShell) verify(ctx context.Context) error {
	hello, err := s.Client.Command(ctx, "hello")
	node := s.node
	matched := false
	for _, line := range strings.Split(hello, "\n") {
		fields := strings.Split(strings.TrimSpace(line), ":")
		if len(fields) >= 3 && fields[0] == "hello" && fields[2] != "" && (node.UID == "" || fields[2] == node.UID) {
			matched = true
		}
	}
	if err != nil || !matched {
		if err != nil {
			return err
		}
		return fmt.Errorf("device identity does not match selected node")
	}
	return nil
}
