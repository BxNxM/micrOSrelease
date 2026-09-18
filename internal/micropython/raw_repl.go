package micropython

import (
	"bytes"
	"context"
	"fmt"
	"time"
)

// Exec executes Python code and returns stdout. A non-empty remote traceback is
// returned as an error.
func (c *Client) Exec(ctx context.Context, code string) ([]byte, error) {
	if c.watchdog {
		code = "if _micros_wdt is not None: _micros_wdt.feed()\n" + code
	}
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	if _, err := c.readUntil(ctx, []byte(">")); err != nil {
		return nil, fmt.Errorf("wait for raw REPL prompt: %w", err)
	}
	if err := c.writeCode(ctx, []byte(code)); err != nil {
		return nil, err
	}
	stdout, err := c.readUntil(ctx, []byte{0x04})
	if err != nil {
		return nil, fmt.Errorf("read MicroPython stdout: %w", err)
	}
	stderr, err := c.readUntil(ctx, []byte{0x04})
	if err != nil {
		return nil, fmt.Errorf("read MicroPython stderr: %w", err)
	}
	stdout = bytes.TrimSuffix(stdout, []byte{0x04})
	stderr = bytes.TrimSuffix(stderr, []byte{0x04})
	if len(bytes.TrimSpace(stderr)) > 0 {
		return nil, fmt.Errorf("MicroPython execution failed: %s", bytes.TrimSpace(stderr))
	}
	return stdout, nil
}

func (c *Client) writeCode(ctx context.Context, code []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.writeAll([]byte{0x05, 'A', 0x01}); err != nil {
		return fmt.Errorf("request raw-paste mode: %w", err)
	}
	response, err := c.readExact(ctx, 2)
	if err != nil {
		return fmt.Errorf("negotiate raw-paste mode: %w", err)
	}
	switch string(response) {
	case "R\x01":
		return c.writeRawPaste(ctx, code)
	case "R\x00":
		return c.writeRaw(ctx, code)
	case "ra":
		if _, err := c.readUntil(ctx, []byte("w REPL; CTRL-B to exit\r\n>")); err != nil {
			return fmt.Errorf("recover unsupported raw-paste response: %w", err)
		}
		return c.writeRaw(ctx, code)
	default:
		return fmt.Errorf("unexpected raw-paste response %q", response)
	}
}

func (c *Client) writeRawPaste(ctx context.Context, code []byte) error {
	windowBytes, err := c.readExact(ctx, 2)
	if err != nil {
		return fmt.Errorf("read raw-paste window: %w", err)
	}
	window := int(windowBytes[0]) | int(windowBytes[1])<<8
	if window <= 0 {
		return fmt.Errorf("invalid raw-paste window %d", window)
	}
	for offset := 0; offset < len(code); {
		end := min(offset+window, len(code))
		if err := c.writeAll(code[offset:end]); err != nil {
			return fmt.Errorf("write raw-paste code: %w", err)
		}
		offset = end
		if offset < len(code) {
			ack, err := c.readExact(ctx, 1)
			if err != nil || ack[0] != 0x01 {
				return fmt.Errorf("raw-paste flow acknowledgement: %q, %w", ack, err)
			}
		}
	}
	if err := c.writeAll([]byte{0x04}); err != nil {
		return err
	}
	ack, err := c.readUntil(ctx, []byte{0x04})
	if err != nil {
		return fmt.Errorf("raw-paste completion acknowledgement: %q, %w", ack, err)
	}
	return nil
}

func (c *Client) writeRaw(ctx context.Context, code []byte) error {
	for len(code) > 0 {
		chunk := min(256, len(code))
		if err := c.writeAll(code[:chunk]); err != nil {
			return fmt.Errorf("write raw REPL code: %w", err)
		}
		code = code[chunk:]
		if len(code) > 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}
	if err := c.writeAll([]byte{0x04}); err != nil {
		return err
	}
	response, err := c.readExact(ctx, 2)
	if err != nil {
		return fmt.Errorf("read raw REPL acknowledgement: %w", err)
	}
	if string(response) != "OK" {
		return fmt.Errorf("unexpected raw REPL acknowledgement %q", response)
	}
	return nil
}
