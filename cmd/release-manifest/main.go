// release-manifest lists application downloads and the latest informational micrOS version.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	selfupdate "github.com/micros/microsctl/internal/network/self_update"
	"github.com/micros/microsctl/internal/usb"
	"gopkg.in/yaml.v3"
)

func main() {
	root := flag.String("root", ".", "repository root")
	output := flag.String("output", "MANIFEST.yaml", "manifest output")
	flag.Parse()
	if err := generate(*root, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate(root, output string) error {
	data, err := os.ReadFile(filepath.Join(root, "MANIFEST.yaml"))
	if err != nil {
		return err
	}
	manifest, err := selfupdate.ParseReleaseManifest(data)
	if err != nil {
		return err
	}
	manifest.Microsctl.Binaries = map[string]selfupdate.ReleaseAsset{}
	for _, platform := range selfupdate.ReleasePlatforms {
		manifest.Microsctl.Binaries[platform] = selfupdate.ReleaseAsset{Path: "dist/" + selfupdate.BinaryName(platform)}
	}

	images, err := usb.Images(os.DirFS(filepath.Join(root, "storage")))
	if err != nil {
		return err
	}
	ordered := usb.BoardImages(images, "")
	if len(ordered) == 0 {
		return fmt.Errorf("no bundled firmware")
	}
	manifest.MicrOS.Version = images[ordered[0]].Version
	if err := manifest.Validate(); err != nil {
		return err
	}
	data, err = yaml.Marshal(manifest)
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(output), ".manifest-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if _, err = temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err = temp.Chmod(0644); err != nil {
		temp.Close()
		return err
	}
	if err = temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), output)
}
