package usb

import (
	"context"
	"fmt"
)

// flashFirmware owns the bootloader connection only. Install and Update each
// manage their own REPL session so resources are copied once per operation.
func (m ReleaseManager) flashFirmware(ctx context.Context, target Target, release preparedRelease, observers ...func([]Stage)) (result Result, err error) {
	config, offset, firmware := release.config, release.offset, release.firmware

	log := []string{"Connect to ESP bootloader", "Erase flash", fmt.Sprintf("Write and verify firmware at %#x", offset), "Reset device"}
	result = Result{Operation: "install", Target: target, Summary: "micrOS firmware installed and verified.", Log: log}
	stages := newStageTracker(log, observers)

	openFlasher := m.OpenFlasher
	if openFlasher == nil {
		openFlasher = func(string, InstallConfig) (deviceFlasher, error) {
			return newESPDeviceFlasher(target.Device, config)
		}
	}
	var flasher deviceFlasher
	if err := stages.run(ctx, 0, func() error {
		var openErr error
		flasher, openErr = openFlasher(target.Device.Port, config)
		if openErr != nil {
			return fmt.Errorf("connect to ESP bootloader on %s before erase (flash was not erased): %w", target.Device.Port, openErr)
		}
		if normalizeChip(flasher.ChipName()) != normalizeChip(config.Chip) {
			return fmt.Errorf("connected chip is %s, but firmware expects %s", flasher.ChipName(), config.Chip)
		}
		_, flashID, err := flasher.FlashID()
		if err != nil {
			return fmt.Errorf("read flash capacity before erase: %w", err)
		}
		capacity := uint8(flashID)
		if capacity < 18 || capacity > 27 {
			return fmt.Errorf("unknown flash capacity in JEDEC ID %04X; flash was not erased", flashID)
		}
		// Include the alignment padding used by the flasher and avoid uint32
		// overflow when an image has a high configured offset.
		end := uint64(offset) + (uint64(len(firmware))+3)&^uint64(3)
		if end > uint64(1)<<capacity {
			return fmt.Errorf("firmware ending at %#x exceeds connected flash capacity %s; flash was not erased", end, flashSizeFromID(flashID))
		}
		return nil
	}); err != nil {
		if flasher != nil {
			flasher.Reset()
			_ = flasher.Close()
		}
		return Result{}, err
	}
	defer func() {
		if closeErr := flasher.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close serial port: %w", closeErr)
		}
	}()

	if config.EraseFlash {
		err = stages.run(ctx, 1, func() error { return flasher.EraseFlash(nil) })
	} else {
		stages.skip(1)
	}
	if err != nil {
		return Result{}, fmt.Errorf("erase flash: %w", err)
	}
	if err = stages.run(ctx, 2, func() error {
		return flasher.FlashImage(firmware, offset, func(current, total int) {
			if total > 0 {
				stages.detail(2, fmt.Sprintf("%d%% transferred", min(100, max(0, int(int64(current)*100/int64(total))))))
			}
		})
	}); err != nil {
		return Result{}, fmt.Errorf("write firmware: %w", err)
	}
	if err = stages.run(ctx, 3, func() error {
		if config.ResetAfterFlash {
			flasher.Reset()
		}
		return nil
	}); err != nil {
		return Result{}, fmt.Errorf("reset device: %w", err)
	}
	return result, nil
}
