package selfupdate

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ExecutableInstaller stages beside the target so replacement stays
// on one filesystem. Path is resolved at startup, before the running file moves.
type ExecutableInstaller struct {
	Path string
}

func (s ExecutableInstaller) Install(ctx context.Context, source io.Reader) (string, error) {
	target, err := filepath.EvalSymlinks(s.Path)
	if err != nil {
		return "", err
	}
	current, err := os.Stat(target)
	if err != nil {
		return "", err
	}
	if !current.Mode().IsRegular() {
		return "", fmt.Errorf("executable is not a regular file")
	}
	file, err := os.CreateTemp(filepath.Dir(target), ".microsctl-update-*")
	if err != nil {
		return "", fmt.Errorf("create update next to executable: %w", err)
	}
	defer os.Remove(file.Name())
	defer file.Close()
	size, err := io.Copy(file, io.LimitReader(&contextReader{ctx: ctx, reader: source}, MaxReleaseSize+1))
	if err != nil {
		return "", fmt.Errorf("download binary: %w", err)
	}
	if size == 0 || size > MaxReleaseSize {
		return "", fmt.Errorf("download is empty or exceeds size limit")
	}
	if err = file.Chmod(current.Mode().Perm() | 0700); err != nil {
		return "", err
	}
	if err = file.Sync(); err != nil {
		return "", err
	}
	if err = file.Close(); err != nil {
		return "", err
	}

	if err = ctx.Err(); err != nil {
		return "", err
	}
	backup, err := backupExecutable(target, current.Mode().Perm())
	if err != nil {
		return "", err
	}
	if err = ctx.Err(); err != nil {
		os.Remove(backup)
		return "", err
	}
	if err = replaceExecutable(file.Name(), target, backup); err != nil {
		return backup, fmt.Errorf("replace executable (backup: %s): %w", backup, err)
	}
	return backup, nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

func backupExecutable(target string, mode os.FileMode) (string, error) {
	source, err := os.Open(target)
	if err != nil {
		return "", err
	}
	defer source.Close()
	backup, err := os.CreateTemp(filepath.Dir(target), ".microsctl-backup-*")
	if err != nil {
		return "", err
	}
	name := backup.Name()
	keep := false
	defer func() {
		backup.Close()
		if !keep {
			os.Remove(name)
		}
	}()
	if _, err = io.Copy(backup, source); err != nil {
		return "", err
	}
	if err = backup.Chmod(mode); err != nil {
		return "", err
	}
	if err = backup.Sync(); err != nil {
		return "", err
	}
	if err = backup.Close(); err != nil {
		return "", err
	}
	keep = true
	return name, nil
}
