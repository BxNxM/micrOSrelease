package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/micros/microsctl/internal/network"
	selfupdate "github.com/micros/microsctl/internal/network/self_update"
	"github.com/micros/microsctl/internal/storage"
	"github.com/micros/microsctl/internal/tui"
	"github.com/micros/microsctl/internal/usb"
	assets "github.com/micros/microsctl/storage"
)

//go:embed MANIFEST.yaml
var releaseManifest []byte

func main() {
	manifest, err := selfupdate.ParseReleaseManifest(releaseManifest)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	version := manifest.Microsctl.Version
	cidr := flag.String("cidr", "", "IPv4 scan range (default: active private /24 networks)")
	dataDir := flag.String("data-dir", "", "persistent data directory (default: platform user config directory)")
	showVersion := flag.Bool("version", false, "print microsctl version and exit")
	listAssets := flag.Bool("list-assets", false, "list embedded framework and module files and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return
	}
	if *listAssets {
		err := fs.WalkDir(assets.Files(), ".", func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() {
				fmt.Println(path)
			}
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "List embedded assets: %v\n", err)
			os.Exit(1)
		}
		return
	}

	store, err := storage.Open(*dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Open storage: %v\n", err)
		os.Exit(1)
	}
	usbManager := usb.ReleaseManager{Assets: assets.Files(), BackupDir: filepath.Join(store.Root, "backups")}
	networkDiscoverer := &network.Service{CIDR: *cidr, Password: os.Getenv("MICROS_PASSWORD"), Store: store}
	executable, err := os.Executable()
	if err == nil {
		executable, err = filepath.EvalSymlinks(executable)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Locate executable: %v\n", err)
		os.Exit(1)
	}
	updater := &selfupdate.SelfUpdater{CurrentVersion: version, URL: manifest.Microsctl.URL, Installer: selfupdate.ExecutableInstaller{Path: executable}}
	firmware := manifest.MicrOS.Version
	model := tui.New(usbManager, networkDiscoverer, tui.WithUpdater(updater, version, firmware))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	final, err := tea.NewProgram(model, tea.WithContext(ctx)).Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "microsctl failed: %v\n", err)
		os.Exit(1)
	}
	if result, ok := final.(interface{ RestartRequested() bool }); ok && result.RestartRequested() {
		if err := selfupdate.Restart(executable, os.Args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "Updated, but restart failed: %v. Run %s again.\n", err, executable)
			os.Exit(1)
		}
	}
}
