package micropython

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"
)

// RuntimeInfo is read from the connected interpreter, never from cached USB metadata.
type RuntimeInfo struct {
	Machine     string `json:"machine"`
	MicroPython string `json:"micropython"`
}

func splitMarkedLines(out []byte, marker string) []string {
	var lines []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, marker) {
			lines = append(lines, strings.TrimPrefix(line, marker))
		}
	}
	return lines
}

func (c *Client) Runtime(ctx context.Context) (RuntimeInfo, error) {
	out, err := c.Exec(ctx, "import os, sys, json\nprint('__MICROS_RUNTIME__'+json.dumps({'machine':os.uname().machine,'micropython':'.'.join(str(v) for v in sys.implementation.version[:3])}))")
	var info RuntimeInfo
	if err != nil {
		return info, err
	}
	line, err := markedLine(out, "__MICROS_RUNTIME__")
	if err != nil {
		return info, err
	}
	err = json.Unmarshal([]byte(line), &info)
	return info, err
}

// FileInfo describes a filesystem entry relative to the device root.
type FileInfo struct {
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	Directory bool   `json:"directory"`
}

// ListFiles includes hidden files and empty directories. It fails closed on
// unsupported entries or oversized inventories instead of silently losing state.
func (c *Client) ListFiles(ctx context.Context) ([]FileInfo, error) {
	var files []FileInfo
	pending := []string{"/"}
	for len(pending) > 0 {
		dir := pending[0]
		pending = pending[1:]
		// Directory stat sizes are unspecified on some MicroPython filesystems.
		code := fmt.Sprintf("import os, json\nfor _e in os.ilistdir(%s):\n _size=0 if (_e[1] & 0xf000)==0x4000 else os.stat(%s+'/'+_e[0])[6]\n print('__MICROS_ENTRY__'+json.dumps([_e[0],_e[1],_size]))", pythonString(dir), pythonString(dir))
		out, err := c.Exec(ctx, code)
		if err != nil {
			return nil, fmt.Errorf("list %s: %w", dir, err)
		}
		entries, err := parseEntries(out, dir)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			files = append(files, entry)
			if len(files) > 10000 {
				return nil, fmt.Errorf("device filesystem exceeds 10000 entries")
			}
			if entry.Directory {
				pending = append(pending, entry.Path)
			}
		}
	}
	return files, nil
}

func parseEntries(out []byte, dir string) ([]FileInfo, error) {
	var entries []FileInfo
	for _, line := range splitMarkedLines(out, "__MICROS_ENTRY__") {
		var row []json.RawMessage
		if err := json.Unmarshal([]byte(line), &row); err != nil || len(row) != 3 {
			return nil, fmt.Errorf("invalid filesystem entry in %s", dir)
		}
		var name string
		var mode int
		var size int64
		if json.Unmarshal(row[0], &name) != nil || json.Unmarshal(row[1], &mode) != nil || json.Unmarshal(row[2], &size) != nil || name == "." || name == ".." || path.Base(name) != name || size < 0 {
			return nil, fmt.Errorf("invalid filesystem entry in %s", dir)
		}
		kind := mode & 0xf000
		if kind != 0x4000 && kind != 0x8000 {
			return nil, fmt.Errorf("unsupported filesystem entry %s/%s", dir, name)
		}
		if kind == 0x4000 {
			size = 0
		}
		entries = append(entries, FileInfo{Path: path.Join(dir, name), Size: size, Directory: kind == 0x4000})
	}
	return entries, nil
}
