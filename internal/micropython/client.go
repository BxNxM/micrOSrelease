// Package micropython implements the small subset of MicroPython's serial raw
// REPL protocol needed for reliable, standalone filesystem updates.
package micropython

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"
)

const (
	defaultTimeout = 10 * time.Second
	maxResponse    = 4 << 20
)

type serialPort interface {
	io.ReadWriteCloser
	SetReadTimeout(time.Duration) error
}

// Client is an active MicroPython raw-REPL session.
type Client struct {
	port     serialPort
	timeout  time.Duration
	inRaw    bool
	watchdog bool
}

// ESP32 watchdogs survive soft resets. Reconfigure and feed the watchdog while
// the application's feeding task is stopped. Reset uses a hardware reset so
// this temporary watchdog does not leak into applications with HA disabled.
func (c *Client) startMaintenance(ctx context.Context) error {
	_, err := c.Exec(ctx, "import sys, machine\n_micros_wdt=machine.WDT(timeout=60000) if sys.platform == 'esp32' else None\nif _micros_wdt is not None: _micros_wdt.feed()")
	if err != nil {
		return fmt.Errorf("prepare maintenance watchdog: %w", err)
	}
	c.watchdog = true
	return nil
}

func newClient(port serialPort, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	return &Client{port: port, timeout: timeout}
}

func (c *Client) enterRawREPL(ctx context.Context) error {
	if err := c.port.SetReadTimeout(100 * time.Millisecond); err != nil {
		return fmt.Errorf("set serial timeout: %w", err)
	}
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	// Immediately after a hardware reset, USB may open before MicroPython can
	// consume Ctrl-C. Retry the interrupt rather than waiting for a prompt from
	// an application that never received it (or repeatedly rebooting that app).
	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("enter raw REPL: %w", err)
		}
		if err := c.writeAll([]byte{'\r', 0x03, 0x03, 0x02}); err != nil {
			return fmt.Errorf("interrupt MicroPython program: %w", err)
		}
		if err := waitContext(ctx, 100*time.Millisecond); err != nil {
			return err
		}
		if err := c.writeAll([]byte{'\r', 0x01}); err != nil {
			return fmt.Errorf("enter raw REPL: %w", err)
		}
		attempt, stop := context.WithTimeout(ctx, time.Second)
		_, err := c.readUntil(attempt, []byte("raw REPL; CTRL-B to exit\r\n>"))
		stop()
		if err == nil {
			break
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("enter raw REPL: %w", err)
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.writeAll([]byte{0x04}); err != nil {
		return fmt.Errorf("soft reset MicroPython: %w", err)
	}
	if _, err := c.readUntil(ctx, []byte("soft reboot\r\n")); err != nil {
		return fmt.Errorf("wait for MicroPython soft reset: %w", err)
	}
	if _, err := c.readUntil(ctx, []byte("raw REPL; CTRL-B to exit\r\n")); err != nil {
		return fmt.Errorf("re-enter raw REPL: %w", err)
	}
	c.inRaw = true
	return nil
}

// Reset interrupts any pending command and restarts the device, clearing the
// maintenance watchdog as well as resuming normal application startup.
func (c *Client) Reset() error {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()
	// Wait for the interpreter, not a driver drain: USB CDC can stop consuming
	// queued bytes during a reboot, leaving tcdrain blocked indefinitely.
	if err := c.writeAll([]byte("\r\x03\x03\x02")); err != nil {
		return err
	}
	if _, err := c.readUntil(ctx, []byte(">>> ")); err != nil {
		return fmt.Errorf("wait for reset prompt: %w", err)
	}
	c.inRaw = false
	// A reset can tear down USB before its final print reaches the host. Finish
	// and acknowledge preparation first, then issue the reset as a separate
	// command. Never turn an arbitrary disconnect into a successful reset.
	if err := c.writeAll([]byte("import machine, os; os.sync(); print('__MICROS_RESET_READY__')\r")); err != nil {
		return fmt.Errorf("prepare device reset: %w", err)
	}
	if _, err := c.readUntil(ctx, []byte("__MICROS_RESET_READY__\r\n")); err != nil {
		return fmt.Errorf("acknowledge reset preparation: %w", err)
	}
	if _, err := c.readUntil(ctx, []byte(">>> ")); err != nil {
		return fmt.Errorf("wait for prepared reset prompt: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := c.writeAll([]byte("machine.reset()\r")); err != nil {
		return fmt.Errorf("send device reset: %w", err)
	}
	c.watchdog = false
	// No response is expected after reset; allow it to start before closing.
	time.Sleep(100 * time.Millisecond)
	return nil
}

// Close leaves raw mode when possible and releases the serial port.
func (c *Client) Close() error {
	if c.inRaw {
		_ = c.writeAll([]byte{'\r', 0x02})
		c.inRaw = false
	}
	return c.port.Close()
}

func (c *Client) readUntil(ctx context.Context, marker []byte) ([]byte, error) {
	buffer := make([]byte, 0, 256)
	one := make([]byte, 1)
	for len(buffer) < maxResponse {
		if err := ctx.Err(); err != nil {
			return buffer, err
		}
		n, err := c.port.Read(one)
		if n > 0 {
			buffer = append(buffer, one[0])
			if bytes.HasSuffix(buffer, marker) {
				return buffer, nil
			}
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return buffer, err
		}
	}
	return buffer, fmt.Errorf("response exceeded %d bytes", maxResponse)
}

func (c *Client) readExact(ctx context.Context, size int) ([]byte, error) {
	data := make([]byte, 0, size)
	buffer := make([]byte, size)
	for len(data) < size {
		if err := ctx.Err(); err != nil {
			return data, err
		}
		n, err := c.port.Read(buffer[:size-len(data)])
		if n > 0 {
			data = append(data, buffer[:n]...)
		}
		if err != nil && !errors.Is(err, io.EOF) {
			return data, err
		}
	}
	return data, nil
}

func (c *Client) writeAll(data []byte) error {
	for len(data) > 0 {
		n, err := c.port.Write(data)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		data = data[n:]
	}
	return nil
}

func (c *Client) withTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, c.timeout)
}

func waitContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
