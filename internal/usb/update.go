package usb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Update preserves the device filesystem, installs and verifies the selected
// release image, then restores the configuration and configured resources over
// MicroPython's serial raw REPL.
func (m ReleaseManager) Update(ctx context.Context, target Target, observers ...func([]Stage)) (result Result, err error) {
	release, err := prepareRelease(m.Assets, target.Image)
	if err != nil {
		return Result{}, err
	}
	config, resources := release.config, release.resources
	if !config.ResetAfterFlash {
		return Result{}, fmt.Errorf("USB update requires reset_after_flash to reconnect to MicroPython")
	}
	if target.Image.Version == "" || target.Image.MicroPython == "" {
		return Result{}, fmt.Errorf("selected firmware requires micrOS and MicroPython version metadata for a safe update")
	}
	target.Device = m.reconnectDevice(ctx, target.Device)
	log := []string{
		"Back up node configuration",
		"Validate runtime and back up filesystem if flashing",
		"Install and verify release firmware",
		"Reconnect to MicroPython REPL",
		"Restore configuration and resources",
		"Reset updated device",
	}
	result = Result{Operation: "update", Target: target, Log: log}
	stages := newStageTracker(log, observers)

	openREPL := m.OpenREPL
	if openREPL == nil {
		openREPL = openMicroPythonREPL
	}
	var session replSession
	resumeOnError := true
	defer func() {
		if session != nil {
			if err != nil && resumeOnError {
				if resetErr := session.Reset(); resetErr != nil {
					err = errors.Join(err, fmt.Errorf("resume unchanged device: %w", resetErr))
				}
			}
			if closeErr := session.Close(); closeErr != nil {
				err = errors.Join(err, fmt.Errorf("close MicroPython REPL: %w", closeErr))
			}
		}
	}()
	var nodeConfig []byte
	var nodeSettings map[string]any
	var nodeVersion, backupPath, configSource string
	var snapshot []savedFile
	alreadyCurrent := false
	if err := stages.run(ctx, 0, func() error {
		session, err = openREPL(ctx, target.Device.Port, config.REPL)
		if err != nil {
			return fmt.Errorf("connect to MicroPython on %s: %w", target.Device.Port, err)
		}
		for _, candidate := range config.REPL.ConfigPaths {
			nodeConfig, err = session.ReadFile(ctx, candidate)
			if err == nil {
				configSource = candidate
				break
			}
		}
		if len(nodeConfig) == 0 {
			return fmt.Errorf("read node_config.json from %s", strings.Join(config.REPL.ConfigPaths, " or "))
		}
		if err := json.Unmarshal(nodeConfig, &nodeSettings); err != nil {
			return fmt.Errorf("validate %s: %w", configSource, err)
		}
		if nodeSettings == nil {
			return fmt.Errorf("validate %s: expected a JSON object", configSource)
		}
		if version, ok := nodeSettings["version"].(string); ok {
			nodeVersion = version
		}
		backupPath, err = archiveNodeConfig(m.BackupDir, nodeSettings, nodeConfig)
		return err
	}); err != nil {
		return Result{}, err
	}

	if err := stages.run(ctx, 1, func() error {
		info, err := session.Runtime(ctx)
		if err != nil {
			return fmt.Errorf("read connected runtime: %w", err)
		}
		chip := runtimeChip(info.Machine)
		if chip == "" || chip != normalizeChip(config.Chip) {
			return fmt.Errorf("connected runtime is %q, but firmware expects %s", info.Machine, config.Chip)
		}
		alreadyCurrent = nodeVersion == target.Image.Version && info.MicroPython == target.Image.MicroPython
		if !alreadyCurrent {
			if m.BackupDir == "" {
				return fmt.Errorf("a backup directory is required before replacing firmware")
			}
			snapshot, err = snapshotDevice(ctx, session)
			if err != nil {
				return err
			}
			configurationSaved := false
			for _, file := range snapshot {
				if file.Path == configSource && bytes.Equal(file.data, nodeConfig) {
					configurationSaved = true
					break
				}
			}
			if !configurationSaved {
				return fmt.Errorf("filesystem backup is missing the original node configuration; flash was not erased")
			}
			identity, _ := nodeSettings["devfid"].(string)
			backupPath, err = archiveDeviceState(m.BackupDir, identity, snapshot)
			if err != nil {
				return fmt.Errorf("archive device filesystem: %w", err)
			}
		}
		return nil
	}); err != nil {
		return Result{}, err
	}
	if alreadyCurrent {
		stages.skip(2)
		stages.skip(3)
	} else {
		if err := stages.run(ctx, 2, func() error {
			resumeOnError = false
			closeErr := session.Close()
			session = nil
			if closeErr != nil {
				return fmt.Errorf("close REPL before flashing: %w", closeErr)
			}
			_, installErr := m.flashFirmware(ctx, target, release)
			return installErr
		}); err != nil {
			return Result{}, updateError("install release firmware", err, backupPath)
		}
		if err := stages.run(ctx, 3, func() error {
			session, result.Target.Device, err = m.reconnectREPL(ctx, target, config, func(detail string) { stages.reconnecting(3, detail) })
			return err
		}); err != nil {
			return Result{}, updateError("reconnect after flashing", err, backupPath)
		}
	}

	if err := stages.run(ctx, 4, func() error {
		resumeOnError = false
		if !alreadyCurrent {
			if err := restoreDeviceState(ctx, session, snapshot, resources, config.REPL); err != nil {
				return err
			}
			nodeSettings["version"] = target.Image.Version
			restoredConfig, err := json.Marshal(nodeSettings)
			if err != nil {
				return fmt.Errorf("encode restored node configuration: %w", err)
			}
			if err := session.WriteFileAtomic(ctx, config.REPL.RestorePath, restoredConfig); err != nil {
				return fmt.Errorf("restore node configuration: %w", err)
			}
		}
		return copyResources(ctx, session, resources)
	}); err != nil {
		return Result{}, updateError("restore device state", err, backupPath)
	}
	if err := stages.run(ctx, 5, session.Reset); err != nil {
		return Result{}, updateError("reset updated device", err, backupPath)
	}
	if alreadyCurrent {
		result.Summary = fmt.Sprintf("micrOS %s is already installed; %d bundled files refreshed and verified.", target.Image.Version, len(resources))
		return result, nil
	}

	fromVersion := nodeVersion
	if fromVersion == "" {
		fromVersion = "unknown"
	}
	result.Summary = fmt.Sprintf("micrOS updated from %s to %s; configuration restored and %d bundled files copied.", fromVersion, target.Image.Version, len(resources))
	return result, nil
}

func updateError(action string, err error, backupPath string) error {
	if backupPath == "" {
		return fmt.Errorf("%s: %w", action, err)
	}
	return fmt.Errorf("%s: %w (device backup: %s)", action, err, backupPath)
}
