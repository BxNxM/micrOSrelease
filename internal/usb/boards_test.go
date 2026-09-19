package usb

import (
	"slices"
	"testing"
)

func TestBoardImagesNewestFirst(t *testing.T) {
	images := []Image{
		{Board: "esp32", Version: "3.9.0-12", MicroPython: "1.29.0"},
		{Board: "esp32c6", Version: "9.0.0"},
		{Board: "esp32", Version: "3.10.0-2", MicroPython: "1.9.0"},
		{Board: "esp32"},
		{Board: "esp32", Version: "3.10.0-10", MicroPython: "1.28.0"},
		{Board: "esp32", Version: "3.10.0-2", MicroPython: "1.28.0"},
		{Board: "esp32", Version: "3.10.0-2", MicroPython: "1.28.0"},
	}
	original := slices.Clone(images)
	for _, tc := range []struct {
		board string
		want  []int
	}{
		{"esp32", []int{4, 5, 6, 2, 0, 3}},
		{"", []int{1, 4, 5, 6, 2, 0, 3}},
		{"esp32s3", nil},
	} {
		if got := BoardImages(images, tc.board); !slices.Equal(got, tc.want) {
			t.Errorf("BoardImages(%q) = %v, want %v", tc.board, got, tc.want)
		}
	}
	if !slices.Equal(images, original) {
		t.Fatal("sorting changed the inventory and invalidated image indices")
	}
}

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
