package micropython

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.bug.st/serial"
)

// Open connects to a serial MicroPython REPL and interrupts the running program.
func Open(ctx context.Context, portName string, baud int, timeout time.Duration) (*Client, error) {
	return openClient(ctx, portName, baud, timeout, true, dialSerial)
}

// OpenForReconnect leaves a booting device alone on failed attempts. Rebooting
// after every failed interrupt can otherwise prevent startup from ever finishing.
func OpenForReconnect(ctx context.Context, portName string, baud int, timeout time.Duration) (*Client, error) {
	return openClient(ctx, portName, baud, timeout, false, dialSerial)
}

func dialSerial(name string, mode *serial.Mode) (serialPort, error) { return serial.Open(name, mode) }

// A driver can block in open/ioctl until the cable is removed. Keep that work
// off the caller, with at most one outstanding connection attempt per endpoint.
// A cancelled attempt must release its handle before another one touches it.
var connectingPorts sync.Map

func openClient(ctx context.Context, portName string, baud int, timeout time.Duration, resumeOnFailure bool, dial func(string, *serial.Mode) (serialPort, error)) (*Client, error) {
	if baud <= 0 {
		baud = 115200
	}
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	key := portName
	// Treat the two macOS endpoint names as one connection.
	if len(key) > len("/dev/tty.") && key[:len("/dev/tty.")] == "/dev/tty." {
		key = "/dev/cu." + key[len("/dev/tty."):]
	}
	value, _ := connectingPorts.LoadOrStore(key, make(chan struct{}, 1))
	gate := value.(chan struct{})
	select {
	case gate <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	type result struct {
		client *Client
		err    error
	}
	ready := make(chan result)
	go func() {
		defer func() { <-gate }()
		port, err := dial(portName, &serial.Mode{BaudRate: baud, DataBits: 8, Parity: serial.NoParity, StopBits: serial.OneStopBit})
		var client *Client
		if err != nil {
			err = fmt.Errorf("open MicroPython serial port %s: %w", portName, err)
		}
		if err == nil {
			// A late open after unplug/replug must not interrupt a new session.
			err = ctx.Err()
			if err == nil {
				client = newClient(port, timeout)
				err = client.enterRawREPL(ctx)
				if err == nil {
					err = client.startMaintenance(ctx)
				}
				if err != nil && resumeOnFailure && ctx.Err() == nil {
					_ = client.Reset()
				}
			}
		}
		if err != nil && port != nil {
			_ = port.Close()
			client = nil
		}
		select {
		case ready <- result{client, err}:
		case <-ctx.Done():
			// Close only: sending reset/raw-mode bytes from an abandoned attempt could
			// interfere with the replacement connection after USB re-enumerates.
			if client != nil {
				_ = client.port.Close()
			}
		}
	}()
	select {
	case r := <-ready:
		return r.client, r.err
	case <-ctx.Done():
		return nil, fmt.Errorf("connect to MicroPython on %s: %w", portName, ctx.Err())
	}
}
