//go:build !windows

package selfupdate

import (
	"os"
	"syscall"
)

func replaceExecutable(staged, target, backup string) error {
	return os.Rename(staged, target)
}

// Restart replaces the process after Bubble Tea restores the terminal.
func Restart(path string, args []string) error {
	return syscall.Exec(path, append([]string{path}, args...), os.Environ())
}
