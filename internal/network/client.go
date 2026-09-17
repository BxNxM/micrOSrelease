package network

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
)

var ansi = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)
var promptPattern = regexp.MustCompile(`(?:^|\n)(?:\[[^\]\n]+\]\s*)*([^\s]+) \$\s*$`)

// Client implements the prompt-framed micrOS socket protocol. Commands on a
// connection must be sequential; callers own and close their client.
type Client struct {
	conn   net.Conn
	prompt string
}

func Dial(ctx context.Context, address, password string) (*Client, error) {
	conn, err := (&net.Dialer{Timeout: time.Second}).DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	c := &Client{conn: conn}
	greeting, err := c.receive(ctx)
	if err == nil && strings.Contains(greeting, "[password]") {
		if password == "" {
			err = errors.New("password required: set MICROS_PASSWORD")
		} else {
			var reply string
			reply, err = c.Command(ctx, password)
			if err == nil && (strings.Contains(reply, "AuthFailed") || strings.Contains(reply, "[password]")) {
				err = errors.New("authentication failed")
			}
		}
	}
	if err == nil {
		host, _, splitErr := net.SplitHostPort(address)
		if splitErr == nil && net.ParseIP(host) == nil && strings.Split(host, ".")[0] != c.prompt {
			err = errors.New("hostname does not match device prompt")
		}
	}
	if err != nil {
		conn.Close()
		return nil, err
	}
	return c, nil
}

func (c *Client) Close() error { return c.conn.Close() }

func (c *Client) Command(ctx context.Context, command string) (string, error) {
	if strings.TrimSpace(command) == "" {
		return "", errors.New("empty command")
	}
	c.deadline(ctx)
	if _, err := fmt.Fprint(c.conn, command); err != nil {
		return "", err
	}
	reply, err := c.receive(ctx)
	if err != nil {
		return "", err
	}
	lines := strings.Split(reply, "\n")
	return strings.TrimSpace(strings.Join(lines[:len(lines)-1], "\n")), nil
}

func (c *Client) deadline(ctx context.Context) {
	deadline := time.Now().Add(3 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	c.conn.SetDeadline(deadline)
}

func (c *Client) receive(ctx context.Context) (string, error) {
	c.deadline(ctx)
	stop := context.AfterFunc(ctx, func() { c.conn.Close() })
	defer stop()
	var data strings.Builder
	buffer := make([]byte, 4096)
	for data.Len() < 1<<20 {
		n, err := c.conn.Read(buffer)
		if n > 0 {
			data.Write(buffer[:n])
		}
		plain := strings.ReplaceAll(ansi.ReplaceAllString(data.String(), ""), "\r", "")
		if strings.Contains(plain, "Bye!") || strings.Contains(plain, "AuthFailed") {
			return "", errors.New("device closed session (busy or authentication failed)")
		}
		match := promptPattern.FindStringSubmatch(plain)
		if match != nil {
			if c.prompt != "" && c.prompt != match[1] {
				return "", errors.New("device prompt changed")
			}
			c.prompt = match[1]
			return plain, nil
		}
		if err != nil {
			return "", fmt.Errorf("receive prompt: %w", err)
		}
	}
	return "", errors.New("response exceeds 1 MiB")
}
