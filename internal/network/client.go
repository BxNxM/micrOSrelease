package network

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"strings"
	"time"
)

var ansi = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)
var promptPattern = regexp.MustCompile(`(?:^|\n)((?:\[[^\]\n]+\][ \t]*)*)([^\s]+) \$[ \t\n]*$`)

// Client implements the prompt-framed micrOS socket protocol. Commands on a
// connection must be sequential; callers own and close their client.
type Client struct {
	conn      net.Conn
	prompt    string
	preprompt string
	timeout   time.Duration
	greeting  string
}

func Dial(ctx context.Context, address, password string) (*Client, error) {
	return dial(ctx, address, password, false)
}

func dial(ctx context.Context, address, password string, interactive bool) (*Client, error) {
	conn, err := (&net.Dialer{Timeout: time.Second}).DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	c := &Client{conn: conn}
	var greeting string
	greeting, err = c.receive(ctx)
	if err == nil {
		c.greeting = promptBody(greeting)
	}
	if err == nil && c.PasswordRequired() {
		if password == "" && !interactive {
			err = errors.New("password required: set MICROS_PASSWORD")
		} else if password != "" {
			var reply string
			reply, err = c.Command(ctx, password)
			if err == nil && (strings.Contains(reply, "AuthFailed") || c.PasswordRequired()) {
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

var ErrSessionClosed = errors.New("session closed")

func (c *Client) Prompt() string         { return c.preprompt + c.prompt + " $ " }
func (c *Client) PasswordRequired() bool { return strings.Contains(c.preprompt, "[password]") }

func (c *Client) Close() error { return c.conn.Close() }

// Command retains the trimmed response convention used by discovery.
func (c *Client) Command(ctx context.Context, command string) (string, error) {
	reply, err := c.StreamCommand(ctx, command, nil)
	return strings.TrimSpace(reply), err
}

// StreamCommand emits cumulative response snapshots as bytes arrive. The final
// snapshot excludes the trailing prompt; response whitespace is preserved.
func (c *Client) StreamCommand(ctx context.Context, command string, emit func(string)) (string, error) {
	if strings.TrimSpace(command) == "" {
		return "", errors.New("empty command")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	stop := context.AfterFunc(ctx, func() { c.conn.Close() })
	defer stop()
	c.deadline(ctx)
	if _, err := fmt.Fprint(c.conn, command); err != nil {
		return "", err
	}
	reply, err := c.receiveStream(ctx, emit)
	if err != nil {
		return reply, err
	}
	return promptBody(reply), nil
}

func promptBody(reply string) string {
	if loc := promptPattern.FindStringIndex(reply); loc != nil {
		end := loc[0]
		// The newline belongs to the response, not the prompt.
		if end < len(reply) && reply[end] == '\n' {
			end++
		}
		return reply[:end]
	}
	return reply
}

func (c *Client) deadline(ctx context.Context) {
	timeout := c.timeout
	if timeout == 0 {
		timeout = 3 * time.Second
	}
	deadline := time.Now().Add(timeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	c.conn.SetDeadline(deadline)
}

func (c *Client) receive(ctx context.Context) (string, error) {
	return c.receiveStream(ctx, nil)
}

func (c *Client) receiveStream(ctx context.Context, emit func(string)) (string, error) {
	return c.receiveStreamLimit(ctx, emit, 1<<20)
}

func (c *Client) receiveStreamLimit(ctx context.Context, emit func(string), responseLimit int) (string, error) {
	c.deadline(ctx)
	stop := context.AfterFunc(ctx, func() { c.conn.Close() })
	defer stop()
	var data strings.Builder
	buffer := make([]byte, 4096)
	for data.Len() <= responseLimit {
		// Shell streams use an inactivity timeout rather than a total reply limit.
		if emit != nil {
			c.deadline(ctx)
		}
		n, err := c.conn.Read(buffer[:min(len(buffer), responseLimit-data.Len()+1)])
		if n > 0 {
			data.Write(buffer[:n])
			if data.Len() > responseLimit {
				plain := strings.ReplaceAll(ansi.ReplaceAllString(data.String()[:responseLimit], ""), "\r", "")
				if emit != nil {
					emit(plain)
				}
				return plain, errors.New("response exceeds 1 MiB")
			}
		}
		plain := strings.ReplaceAll(ansi.ReplaceAllString(data.String(), ""), "\r", "")
		if emit != nil {
			if promptPattern.MatchString(plain) {
				emit(promptBody(plain))
			} else {
				emit(plain)
			}
		}
		match := promptPattern.FindStringSubmatch(plain)
		if c.PasswordRequired() && (match != nil || errors.Is(err, io.EOF)) {
			for _, line := range strings.Split(plain, "\n") {
				if strings.TrimSpace(line) == "AuthFailed" {
					return plain, errors.New("authentication failed")
				}
			}
		}
		if match != nil {
			if c.prompt != "" && c.prompt != match[2] {
				return plain, errors.New("device prompt changed")
			}
			c.prompt = match[2]
			c.preprompt = strings.TrimSpace(match[1])
			if c.preprompt != "" {
				c.preprompt += " "
			}
			return plain, nil
		}
		if err != nil {
			// A read boundary is not a message boundary: ordinary output can
			// contain these words. Classify a close only once the peer closes.
			if errors.Is(err, io.EOF) {
				last := strings.TrimSpace(plain)
				if i := strings.LastIndex(last, "\n"); i >= 0 {
					last = strings.TrimSpace(last[i+1:])
				}
				switch last {
				case "Connection is busy. Bye!":
					return plain, errors.New("server busy")
				case "Bye!":
					return plain, ErrSessionClosed
				}
			}
			return plain, fmt.Errorf("receive prompt: %w", err)
		}
	}
	return strings.ReplaceAll(ansi.ReplaceAllString(data.String(), ""), "\r", ""), errors.New("response exceeds 1 MiB")
}
