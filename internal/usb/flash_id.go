package usb

import (
	"errors"
	"fmt"
)

type flashIDReader interface {
	ChipName() string
	FlashID() (uint8, uint16, error)
	ReadRegister(uint32) (uint32, error)
	WriteRegister(uint32, uint32) error
}

// readFlashID works around espflasher v0.8.1 treating the original ESP32's
// receive length like an ESP8266. ESP32 needs SPI_MISO_DLEN at base + 0x2C;
// without it, RDID returns incomplete data and stale bits from the data buffer.
// Register layout: https://github.com/espressif/esptool/blob/master/esptool/targets/esp32.py
func readFlashID(f flashIDReader) (manufacturer uint8, id uint16, err error) {
	if normalizeChip(f.ChipName()) != "esp32" {
		return f.FlashID()
	}
	const misoLength = 0x3FF4202C
	const dataBuffer = 0x3FF42080
	previous, err := f.ReadRegister(misoLength)
	if err != nil {
		return 0, 0, fmt.Errorf("read SPI receive length: %w", err)
	}
	defer func() {
		if restoreErr := f.WriteRegister(misoLength, previous); restoreErr != nil {
			err = errors.Join(err, fmt.Errorf("restore SPI receive length: %w", restoreErr))
		}
	}()
	if err := f.WriteRegister(misoLength, 23); err != nil { // 24-bit JEDEC response
		return 0, 0, fmt.Errorf("set SPI receive length: %w", err)
	}
	if err := f.WriteRegister(dataBuffer, 0); err != nil {
		return 0, 0, fmt.Errorf("clear SPI receive buffer: %w", err)
	}
	return f.FlashID()
}
