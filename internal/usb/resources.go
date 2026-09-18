package usb

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/micros/microsctl/internal/micropython"
)

func releaseResources(files fs.FS, config InstallConfig) ([]UpdateResource, error) {
	resources, err := bundledResources(files, config.Platform)
	if err != nil {
		return nil, err
	}
	byTarget := make(map[string]int, len(resources)+len(config.REPL.Resources))
	for index, resource := range resources {
		byTarget[resource.Target] = index
	}
	for _, resource := range config.REPL.Resources {
		if index, ok := byTarget[resource.Target]; ok {
			resources[index] = resource
			continue
		}
		byTarget[resource.Target] = len(resources)
		resources = append(resources, resource)
	}
	sort.SliceStable(resources, func(i, j int) bool {
		return path.Base(resources[i].Target) != "main.py" && path.Base(resources[j].Target) == "main.py"
	})
	return resources, nil
}

func bundledResources(files fs.FS, platform string) ([]UpdateResource, error) {
	var resources []UpdateResource
	err := fs.WalkDir(files, "modules", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			if name == "modules" && errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if entry.IsDir() || name == "modules/README.md" {
			return nil
		}
		base := path.Base(name)
		if path.Dir(name) == "modules/modules" && strings.HasPrefix(base, "IO_") &&
			(path.Ext(base) == ".mpy" || path.Ext(base) == ".py") &&
			base != "IO_"+platform+".mpy" && base != "IO_"+platform+".py" {
			return nil
		}
		target := "/" + strings.TrimPrefix(name, "modules/")
		if !validDevicePath(target) {
			return fmt.Errorf("invalid bundled resource target %q", target)
		}
		resources = append(resources, UpdateResource{Source: name, Target: target})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("catalog bundled resources: %w", err)
	}
	return resources, nil
}

type preparedResource struct {
	target string
	data   []byte
}

// Read all payloads before touching the board so a missing configured source
// cannot first be discovered after its flash has been erased.
func prepareResources(files fs.FS, config InstallConfig) ([]preparedResource, error) {
	resources, err := releaseResources(files, config)
	if err != nil {
		return nil, err
	}
	prepared := make([]preparedResource, 0, len(resources))
	for _, resource := range resources {
		data, err := fs.ReadFile(files, resource.Source)
		if err != nil {
			return nil, fmt.Errorf("read bundled resource %s: %w", resource.Source, err)
		}
		if len(data) > micropython.MaxFileSize {
			return nil, fmt.Errorf("bundled resource %s exceeds transfer limit of %d bytes", resource.Source, micropython.MaxFileSize)
		}
		prepared = append(prepared, preparedResource{target: resource.Target, data: data})
	}
	return prepared, nil
}

func copyResources(ctx context.Context, session replSession, resources []preparedResource) error {
	for _, resource := range resources {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := session.WriteFileAtomic(ctx, resource.target, resource.data); err != nil {
			return fmt.Errorf("copy resource %s: %w", resource.target, err)
		}
	}
	return nil
}
