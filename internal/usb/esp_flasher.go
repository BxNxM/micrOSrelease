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

func newESPDeviceFlasher(port string, config InstallConfig) (deviceFlasher, error) {
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
		options.ResetMode = espflasher.ResetAuto
	default:
		return nil, fmt.Errorf("unsupported reset mode %q", config.ResetMode)
	}
	flasher, err := espflasher.New(port, options)
	if err != nil {
		return nil, err
	}
	return &espDeviceFlasher{flasher: flasher}, nil
}

func (f *espDeviceFlasher) ChipName() string { return f.flasher.ChipName() }

func (f *espDeviceFlasher) FlashID() (uint8, uint16, error) { return f.flasher.FlashID() }

func (f *espDeviceFlasher) EraseFlash(progress func(current, total int)) error {
	return f.flasher.EraseFlash(espflasher.ProgressFunc(progress))
}

func (f *espDeviceFlasher) FlashImage(data []byte, offset uint32, progress func(current, total int)) error {
	return f.flasher.FlashImage(data, offset, espflasher.ProgressFunc(progress))
}

func (f *espDeviceFlasher) Reset() { f.flasher.Reset() }

func (f *espDeviceFlasher) Close() error { return f.flasher.Close() }
