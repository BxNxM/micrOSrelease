package usb

import (
	"context"
	"fmt"
)

// Install writes and verifies firmware, then copies bundled resources through
// MicroPython's raw REPL before starting the installed application.
func (m ReleaseManager) Install(ctx context.Context, target Target, observers ...func([]Stage)) (result Result, err error) {
	release, err := prepareRelease(m.Assets, target.Image)
	if err != nil {
		return Result{}, err
	}
	config, resources := release.config, release.resources
	if len(resources) == 0 {
		return m.flashFirmware(ctx, target, release, observers...)
	}
	if !config.ResetAfterFlash {
		return Result{}, fmt.Errorf("resource installation requires reset_after_flash to start MicroPython")
	}

	target.Device = m.reconnectDevice(ctx, target.Device)
	log := []string{"Install and verify release firmware", "Reconnect to MicroPython REPL", "Copy and verify bundled resources", "Reset installed device"}
	result = Result{Operation: "install", Target: target, Log: log}
	stages := newStageTracker(log, observers)
	if err := stages.run(ctx, 0, func() error {
		_, err := m.flashFirmware(ctx, target, release)
		return err
	}); err != nil {
		return Result{}, err
	}
	var session replSession
	if err := stages.run(ctx, 1, func() error {
		session, result.Target.Device, err = m.reconnectREPL(ctx, target, config, func(detail string) { stages.reconnecting(1, detail) })
		return err
	}); err != nil {
		return Result{}, fmt.Errorf("reconnect after flashing: %w", err)
	}
	defer func() {
		if closeErr := session.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close MicroPython REPL: %w", closeErr)
		}
	}()
	if err := stages.run(ctx, 2, func() error { return copyResources(ctx, session, resources) }); err != nil {
		return Result{}, err
	}
	if err := stages.run(ctx, 3, session.Reset); err != nil {
		return Result{}, fmt.Errorf("reset installed device: %w", err)
	}
	result.Summary = fmt.Sprintf("micrOS firmware installed; %d bundled files copied and verified.", len(resources))
	return result, nil
}
