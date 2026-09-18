package usb

import (
	"context"
	"fmt"
	"net"
	"strings"

	"tinygo.org/x/espflasher/pkg/espflasher"
)

type deviceProbe interface {
	ChipName() string
	FlashID() (uint8, uint16, error)
	MAC() (net.HardwareAddr, error)
	ChipRevision() (int, int, error)
	ChipFeatures() ([]string, error)
	Reset()
	Close() error
}

type espDeviceProbe struct {
	flasher *espflasher.Flasher
}

func newESPDeviceProbe(port string) (deviceProbe, error) {
	options := espflasher.DefaultOptions()
	options.ResetMode = espflasher.ResetAuto
	options.SkipStub = true
	flasher, err := espflasher.New(port, options)
	if err != nil {
		return nil, err
	}
	return &espDeviceProbe{flasher: flasher}, nil
}

func (p *espDeviceProbe) ChipName() string { return p.flasher.ChipName() }

func (p *espDeviceProbe) FlashID() (uint8, uint16, error) { return p.flasher.FlashID() }

func (p *espDeviceProbe) MAC() (net.HardwareAddr, error) { return p.flasher.MAC() }

func (p *espDeviceProbe) ChipRevision() (int, int, error) {
	revision, err := p.flasher.ChipRevision()
	return revision.Major, revision.Minor, err
}

func (p *espDeviceProbe) ChipFeatures() ([]string, error) { return p.flasher.ChipFeatures() }

func (p *espDeviceProbe) Reset() { p.flasher.Reset() }

func (p *espDeviceProbe) Close() error { return p.flasher.Close() }

// Probe identifies an ESP device and enriches the supplied serial endpoint.
// Entering the ROM bootloader resets the board, so callers should invoke this
// only as an explicit user action rather than as part of passive discovery.
func (m ReleaseManager) Probe(ctx context.Context, device Device) (Device, error) {
	if strings.TrimSpace(device.Port) == "" {
		return device, fmt.Errorf("USB device port is required")
	}
	if err := ctx.Err(); err != nil {
		return device, err
	}

	opener := m.OpenProbe
	if opener == nil {
		opener = newESPDeviceProbe
	}
	probe, err := opener(device.Port)
	if err != nil {
		return device, fmt.Errorf("connect to ESP bootloader on %s: %w", device.Port, err)
	}
	defer probe.Close() //nolint:errcheck -- identification result is more useful than a close error
	defer probe.Reset()

	info := &DeviceInfo{Chip: probe.ChipName()}
	if info.Chip == "" {
		return device, fmt.Errorf("device on %s did not report a chip type", device.Port)
	}

	if manufacturer, flashID, flashErr := probe.FlashID(); flashErr != nil {
		info.Warnings = append(info.Warnings, "flash: "+flashErr.Error())
	} else {
		info.FlashID = fmt.Sprintf("%02X:%04X", manufacturer, flashID)
		info.FlashSize = flashSizeFromID(flashID)
	}
	if mac, macErr := probe.MAC(); macErr != nil {
		info.Warnings = append(info.Warnings, "MAC: "+macErr.Error())
	} else {
		info.MAC = mac.String()
	}
	if major, minor, revisionErr := probe.ChipRevision(); revisionErr != nil {
		info.Warnings = append(info.Warnings, "revision: "+revisionErr.Error())
	} else {
		info.Revision = fmt.Sprintf("v%d.%d", major, minor)
	}
	if features, featuresErr := probe.ChipFeatures(); featuresErr != nil {
		info.Warnings = append(info.Warnings, "features: "+featuresErr.Error())
	} else {
		info.Features = append([]string(nil), features...)
	}
	if err := ctx.Err(); err != nil {
		return device, err
	}
	device.Info = info
	return device, nil
}

func flashSizeFromID(flashID uint16) string {
	capacity := uint8(flashID)
	if capacity < 18 || capacity > 27 {
		return ""
	}
	bytes := uint64(1) << capacity
	if bytes < 1024*1024 {
		return fmt.Sprintf("%d KB", bytes/1024)
	}
	return fmt.Sprintf("%d MB", bytes/(1024*1024))
}
