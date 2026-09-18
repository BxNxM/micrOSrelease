package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/micros/microsctl/internal/network"
	"github.com/micros/microsctl/internal/storage"
	"github.com/micros/microsctl/internal/tui"
	"github.com/micros/microsctl/internal/usb"
	assets "github.com/micros/microsctl/storage"
)

func main() {
	demoDelay := flag.Duration("demo-delay", 1200*time.Millisecond, "simulated feature latency")
	cidr := flag.String("cidr", "", "IPv4 scan range (default: active private /24 networks)")
	dataDir := flag.String("data-dir", "", "persistent data directory (default: platform user config directory)")
	listAssets := flag.Bool("list-assets", false, "list embedded framework and module files and exit")
	flag.Parse()
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
	usbManager := usb.DummyManager{Delay: *demoDelay, Assets: assets.Files()}
	networkDiscoverer := &network.Service{CIDR: *cidr, Password: os.Getenv("MICROS_PASSWORD"), Store: store}
	model := tui.New(usbManager, networkDiscoverer)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if _, err := tea.NewProgram(model, tea.WithContext(ctx)).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "microsctl failed: %v\n", err)
		os.Exit(1)
	}
}
