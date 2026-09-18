package usb

import (
	"errors"
	"testing"
)

type fakeFlashIDReader struct {
	chip               string
	length, buffer     uint32
	readErr, flashErr  error
	failWrite          int
	writes, flashCalls int
}

func (f *fakeFlashIDReader) ChipName() string { return f.chip }
func (f *fakeFlashIDReader) ReadRegister(addr uint32) (uint32, error) {
	if addr != 0x3FF4202C {
		panic("unexpected register")
	}
	return f.length, f.readErr
}
func (f *fakeFlashIDReader) WriteRegister(addr, value uint32) error {
	f.writes++
	if f.writes == f.failWrite {
		return errors.New("register write failed")
	}
	switch addr {
	case 0x3FF4202C:
		f.length = value
	case 0x3FF42080:
		f.buffer = value
	default:
		panic("unexpected register")
	}
	return nil
}
func (f *fakeFlashIDReader) FlashID() (uint8, uint16, error) {
	f.flashCalls++
	if f.chip == "ESP32" && (f.length != 23 || f.buffer != 0) {
		return 0, 0x8B95, nil // Incomplete response / stale buffer from the regression.
	}
	return 0x20, 0x4016, f.flashErr
}

func TestReadFlashIDCorrectsESP32ReceiveLength(t *testing.T) {
	f := &fakeFlashIDReader{chip: "ESP32", length: 7, buffer: 0x8B9500}
	manufacturer, id, err := readFlashID(f)
	if err != nil || manufacturer != 0x20 || flashSizeFromID(id) != "4 MB" {
		t.Fatalf("flash = %02X:%04X, error %v", manufacturer, id, err)
	}
	if f.length != 7 {
		t.Fatal("SPI receive length was not restored")
	}
}

func TestReadFlashIDLeavesOtherChipsUntouched(t *testing.T) {
	for _, chip := range []string{"ESP32-C6", "ESP32-S3", "ESP8266"} {
		f := &fakeFlashIDReader{chip: chip, readErr: errors.New("must not access ESP32 registers")}
		_, id, err := readFlashID(f)
		if err != nil || id != 0x4016 || f.writes != 0 || f.flashCalls != 1 {
			t.Fatalf("%s: id=%04X, error=%v, writes=%d", chip, id, err, f.writes)
		}
	}
}

func TestReadFlashIDPropagatesFailures(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		readErr, flashErr     error
		failWrite, flashCalls int
	}{
		{name: "read", readErr: errors.New("read failed")},
		{name: "set length", failWrite: 1},
		{name: "clear buffer", failWrite: 2},
		{name: "flash command", flashErr: errors.New("flash read failed"), flashCalls: 1},
		{name: "restore", failWrite: 3, flashCalls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeFlashIDReader{chip: "ESP32", length: 7, readErr: tc.readErr, flashErr: tc.flashErr, failWrite: tc.failWrite}
			_, _, err := readFlashID(f)
			if err == nil || f.flashCalls != tc.flashCalls {
				t.Fatalf("error=%v, flash calls=%d", err, f.flashCalls)
			}
			if tc.failWrite != 3 && f.length != 7 {
				t.Fatal("SPI receive length was not restored after failure")
			}
		})
	}
}
