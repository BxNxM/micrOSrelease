package usb

import (
	"io/fs"
	"path"
	"testing"

	assets "github.com/micros/microsctl/storage"
)

func TestValidateInstallConfig(t *testing.T) {
	config := InstallConfig{Version: 1, Protocol: "esp-rom", Chip: "esp32-s3", InitialBaud: 115200, FlashBaud: 460800, FlashOffset: "0x1000", ResetMode: "usb-jtag"}
	offset, err := validateInstallConfig(config)
	if err != nil || offset != 0x1000 {
		t.Fatalf("validateInstallConfig() = %#x, %v", offset, err)
	}

	config.Chip = "rp2040"
	if _, err := validateInstallConfig(config); err == nil {
		t.Fatal("unsupported chip must fail validation")
	}
}

func TestBundledInstallConfigsAreValid(t *testing.T) {
	files := assets.Files()
	count := 0
	err := fs.WalkDir(files, "frameworks", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || entry.Name() != "install.json" {
			return nil
		}
		count++
		_, _, err = loadInstallConfig(files, Image{Path: path.Join(path.Dir(name), "firmware.bin")})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("no bundled board configs found")
	}
	images, err := Images(files)
	if err != nil {
		t.Fatal(err)
	}
	for _, image := range images {
		if _, err := prepareRelease(files, image); err != nil {
			t.Errorf("firmware %s cannot be prepared for release: %v", image.Path, err)
		}
	}
}
