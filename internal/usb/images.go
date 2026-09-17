package usb

import (
	"io/fs"
	"path"
	"strings"
)

// Images catalogs embedded firmware without reading binary contents or opening hardware.
func Images(files fs.FS) ([]Image, error) {
	var images []Image
	if files == nil {
		return images, nil
	}
	err := fs.WalkDir(files, "frameworks", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		ext := strings.ToLower(path.Ext(name))
		if ext != ".bin" && ext != ".uf2" {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		image := Image{Path: name, Name: entry.Name(), Size: info.Size(), Board: "unknown", Release: strings.HasPrefix(entry.Name(), "micrOS-")}
		if image.Release {
			parts := strings.SplitN(strings.TrimSuffix(strings.TrimPrefix(entry.Name(), "micrOS-"), path.Ext(name)), "-", 3)
			if len(parts) == 3 {
				image.Board, image.MicroPython, image.Version = parts[0], parts[1], parts[2]
			}
		}
		// The board directory also supports firmware with nonstandard filenames.
		if relative := strings.TrimPrefix(name, "frameworks/"); strings.Contains(relative, "/") {
			image.Board = strings.SplitN(relative, "/", 2)[0]
		}
		images = append(images, image)
		return nil
	})
	return images, err
}
