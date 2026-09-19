package usb

import (
	"fmt"

	"tinygo.org/x/espflasher/pkg/espflasher"
)

type deviceFlasher interface {
	ChipName() string
	FlashID() (uint8, uint16, error)
	EraseFlash(func(current, total int)) error
	FlashImage([]byte, uint32, func(current, total int)) error
	Reset()
	Close() error
}

type espDeviceFlasher struct {
	flasher *espflasher.Flasher
}

func flasherOptions(device Device, config InstallConfig) (*espflasher.FlasherOptions, error) {
	options := espflasher.DefaultOptions()
	options.BaudRate = config.InitialBaud
	options.FlashBaudRate = config.FlashBaud
	options.Compress = config.Compress
	options.FlashMode = config.FlashMode
	options.FlashFreq = config.FlashFrequency
	options.FlashSize = config.FlashSize
	switch config.ResetMode {
	case "default":
		options.ResetMode = espflasher.ResetDefault
	case "usb-jtag":
		options.ResetMode = espflasher.ResetUSBJTAG
	case "no-reset":
		options.ResetMode = espflasher.ResetNoReset
	case "auto":
		options.ResetMode = automaticResetMode(device)
		// These chips always use a UART bridge, including Windows COM ports
		// where the name alone cannot identify the transport.
		if chip := normalizeChip(config.Chip); chip == "esp32" || chip == "esp8266" {
			options.ResetMode = espflasher.ResetDefault
		}
	default:
		return nil, fmt.Errorf("unsupported reset mode %q", config.ResetMode)
	}
	return options, nil
}

func newESPDeviceFlasher(device Device, config InstallConfig) (deviceFlasher, error) {
	options, err := flasherOptions(device, config)
	if err != nil {
		return nil, err
	}
	flasher, err := espflasher.New(device.Port, options)
	if err != nil {
		return nil, err
	}
	return &espDeviceFlasher{flasher: flasher}, nil
}

func (f *espDeviceFlasher) ChipName() string { return f.flasher.ChipName() }

func (f *espDeviceFlasher) FlashID() (uint8, uint16, error) { return readFlashID(f.flasher) }

func (f *espDeviceFlasher) EraseFlash(progress func(current, total int)) error {
	return f.flasher.EraseFlash(espflasher.ProgressFunc(progress))
}

func (f *espDeviceFlasher) FlashImage(data []byte, offset uint32, progress func(current, total int)) error {
	return f.flasher.FlashImage(data, offset, espflasher.ProgressFunc(progress))
}

func (f *espDeviceFlasher) Reset() { f.flasher.Reset() }

func (f *espDeviceFlasher) Close() error { return f.flasher.Close() }
