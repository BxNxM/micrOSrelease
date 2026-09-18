package usb

import "testing"

func TestLatestBoardImageUsesNumericMicrOSVersion(t *testing.T) {
	images := []Image{
		{Board: "esp32", Version: "3.10.0-1", MicroPython: "1.27.0"},
		{Board: "esp32c6", Version: "9.0.0"},
		{Board: "esp32", Version: "3.9.0-12", MicroPython: "1.29.0"},
		{Board: "esp32", Version: "3.10.0-2", MicroPython: "1.28.0"},
	}
	index, found := LatestBoardImage(images, "esp32")
	if !found || index != 3 {
		t.Fatalf("latest index = %d, found = %v", index, found)
	}
	if _, found := LatestBoardImage(images, "esp32s3"); found {
		t.Fatal("reported firmware for a missing board")
	}
}

func TestLatestBoardImageBreaksMicrOSTieWithMicroPythonVersion(t *testing.T) {
	images := []Image{
		{Board: "esp32", Version: "3.10.0-2", MicroPython: "1.9.0"},
		{Board: "esp32", Version: "3.10.0-2", MicroPython: "1.28.0"},
	}
	index, found := LatestBoardImage(images, "esp32")
	if !found || index != 1 {
		t.Fatalf("latest index = %d, found = %v", index, found)
	}
}
