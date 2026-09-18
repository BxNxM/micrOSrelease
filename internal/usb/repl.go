package usb

import (
	"context"
	"fmt"
	"time"

	"github.com/micros/microsctl/internal/micropython"
)

type replSession interface {
	Runtime(context.Context) (micropython.RuntimeInfo, error)
	ListFiles(context.Context) ([]micropython.FileInfo, error)
	MkdirAll(context.Context, string) error
	ReadFile(context.Context, string) ([]byte, error)
	WriteFileAtomic(context.Context, string, []byte) error
	Reset() error
	Close() error
}

func openReconnectREPL(ctx context.Context, port string, config REPLConfig) (replSession, error) {
	return micropython.OpenForReconnect(ctx, port, config.Baud, time.Duration(config.ConnectTimeoutSeconds)*time.Second)
}

func openMicroPythonREPL(ctx context.Context, port string, config REPLConfig) (replSession, error) {
	return micropython.Open(ctx, port, config.Baud, time.Duration(config.ConnectTimeoutSeconds)*time.Second)
}

// reconnectDevice captures passive USB identity before flashing can remove the port.
func (m ReleaseManager) reconnectDevice(ctx context.Context, device Device) Device {
	if device.USBSerial != "" {
		return device
	}
	discover := m.Discover
	if discover == nil {
		discover = discoverDevices
	}
	scanCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	devices, err := discover(scanCtx)
	if err == nil {
		for _, candidate := range devices {
			if samePort(candidate.Port, device.Port) {
				candidate.Info = device.Info
				return candidate
			}
		}
	}
	return device
}

// reconnectREPL keeps the operation and its in-memory restore plan alive while
// USB is absent. A positive manifest timeout remains available for automation;
// the default is to wait until the caller cancels.
func (m ReleaseManager) reconnectREPL(ctx context.Context, target Target, config InstallConfig, progress func(string)) (replSession, Device, error) {
	if config.REPL.ReconnectTimeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(config.REPL.ReconnectTimeoutSeconds)*time.Second)
		defer cancel()
	}
	open := m.OpenREPL
	if open == nil {
		open = openReconnectREPL
	}
	discover := m.Discover
	if discover == nil {
		discover = discoverDevices
	}
	lastMessage := ""
	report := func(message string) {
		if progress != nil && message != lastMessage {
			progress(message)
			lastMessage = message
		}
	}
	var lastErr error
	for {
		if err := ctx.Err(); err != nil {
			return nil, target.Device, fmt.Errorf("MicroPython reconnect stopped (last error: %v): %w", lastErr, err)
		}
		report("Waiting for MicroPython. You may unplug and reconnect this board; this operation will continue automatically.")
		scanCtx, stopScan := context.WithTimeout(ctx, 5*time.Second)
		devices, scanErr := discover(scanCtx)
		stopScan()
		candidates := reconnectCandidates(target.Device, devices)
		if target.Device.USBSerial == "" && len(candidates) == 0 {
			// With no stable identity, never guess another port (or another board).
			// Also permits explicitly supplied ports outside the discovery filters.
			candidates = []Device{target.Device}
		}
		if scanErr != nil {
			lastErr = scanErr
		}
		for _, candidate := range candidates {
			report("Connecting on " + candidate.Port + ". If it stays here, unplug and reconnect USB; keep this operation open.")
			attempt, cancel := context.WithTimeout(ctx, time.Duration(config.REPL.withDefaults().ConnectTimeoutSeconds)*time.Second)
			session, err := open(attempt, candidate.Port, config.REPL)
			if err == nil {
				var info micropython.RuntimeInfo
				info, err = session.Runtime(attempt)
				if err == nil && runtimeChip(info.Machine) != normalizeChip(config.Chip) {
					err = fmt.Errorf("reconnected chip %q does not match %s", info.Machine, config.Chip)
				}
				if err == nil && target.Image.MicroPython != "" && info.MicroPython != target.Image.MicroPython {
					err = fmt.Errorf("reconnected MicroPython %s does not match %s", info.MicroPython, target.Image.MicroPython)
				}
				if err == nil {
					err = attempt.Err()
				}
				if err == nil {
					cancel()
					candidate.Info = target.Device.Info
					return session, candidate, nil
				}
				// No files are written until the returned interpreter has been validated.
				_ = session.Reset()
				_ = session.Close()
			}
			cancel()
			lastErr = err
		}
		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
}
