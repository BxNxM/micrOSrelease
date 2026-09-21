package selfupdate

import (
	"fmt"
	"os"
	"os/exec"
)

func replaceExecutable(staged, target, backup string) error {
	// Windows permits moving the running image but cannot overwrite it in place.
	// The durable copy remains available even if restoring the moved image fails.
	moved := backup + ".running.exe"
	if err := os.Rename(target, moved); err != nil {
		return err
	}
	if err := os.Rename(staged, target); err != nil {
		if restoreErr := os.Rename(moved, target); restoreErr != nil {
			return fmt.Errorf("install: %v; restore: %v; recovery copy: %s", err, restoreErr, backup)
		}
		return err
	}
	// The old process may keep this image locked until it exits.
	_ = os.Remove(moved)
	return nil
}

func Restart(path string, args []string) error {
	command := exec.Command(path, args...)
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	command.Env = os.Environ()
	if err := command.Start(); err != nil {
		return err
	}
	return command.Process.Release()
}
