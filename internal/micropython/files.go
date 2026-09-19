package micropython

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"strings"
)

const (
	fileMarker = "__MICROS_FILE__"
	hashMarker = "__MICROS_SHA256__"
	chunkSize  = 256
	// MaxFileSize bounds transfers and is also checked before destructive work.
	MaxFileSize = 1 << 20
)

// ReadFile downloads a file and caps its size to protect host and device memory.
func (c *Client) ReadFile(ctx context.Context, name string) ([]byte, error) {
	return c.readFile(ctx, name, true)
}

func (c *Client) readFile(ctx context.Context, name string, verify bool) ([]byte, error) {
	quoted := pythonString(name)
	data := make([]byte, 0, chunkSize)
	for offset := 0; offset <= MaxFileSize; offset += chunkSize {
		code := fmt.Sprintf("import ubinascii\n_f=open(%s,'rb')\n_f.seek(%d)\n_d=_f.read(%d)\n_f.close()\nprint('%s'+ubinascii.hexlify(_d).decode())", quoted, offset, chunkSize, fileMarker)
		output, err := c.Exec(ctx, code)
		if err != nil {
			return nil, fmt.Errorf("read %s at byte %d: %w", name, offset, err)
		}
		encoded, err := markedLine(output, fileMarker)
		if err != nil {
			return nil, fmt.Errorf("read %s at byte %d: %w", name, offset, err)
		}
		chunk, err := hex.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("decode %s at byte %d: %w", name, offset, err)
		}
		if len(data)+len(chunk) > MaxFileSize {
			return nil, fmt.Errorf("%s exceeds %d bytes", name, MaxFileSize)
		}
		data = append(data, chunk...)
		if len(chunk) < chunkSize {
			if !verify {
				return data, nil
			}
			// A backup must not authorize erase after a silently corrupted read.
			digest, err := c.fileSHA256(ctx, name)
			if err != nil {
				return nil, fmt.Errorf("verify downloaded %s: %w", name, err)
			}
			expected := sha256.Sum256(data)
			if !bytes.Equal(digest, expected[:]) {
				return nil, fmt.Errorf("verify downloaded %s: SHA-256 mismatch", name)
			}
			return data, nil
		}
	}
	return nil, fmt.Errorf("%s exceeds %d bytes", name, MaxFileSize)
}

// WriteFileAtomic uploads, reads back, and atomically replaces a device file.
func (c *Client) WriteFileAtomic(ctx context.Context, name string, data []byte) error {
	if len(data) > MaxFileSize {
		return fmt.Errorf("%s exceeds %d bytes", name, MaxFileSize)
	}
	if err := c.MkdirAll(ctx, path.Dir(name)); err != nil {
		return err
	}
	// Keep scratch files beside the destination, with names belonging only to
	// this transfer. Never overwrite a user's file or an earlier recovery copy.
	scratch := path.Join(path.Dir(name), ".micros-"+rand.Text())
	temporary, backup := scratch+".tmp", scratch+".bak"
	if _, err := c.Exec(ctx, createUploadScript(temporary, backup)); err != nil {
		return fmt.Errorf("create temporary file for %s: %w", name, err)
	}
	for offset := 0; offset < len(data); offset += chunkSize {
		end := min(offset+chunkSize, len(data))
		code := fmt.Sprintf("import ubinascii\n_f.write(ubinascii.unhexlify('%s'))", hex.EncodeToString(data[offset:end]))
		if _, err := c.Exec(ctx, code); err != nil {
			_, _ = c.Exec(context.Background(), "_f.close()")
			return fmt.Errorf("upload %s at byte %d: %w", name, offset, err)
		}
	}
	if _, err := c.Exec(ctx, "_f.close()"); err != nil {
		return fmt.Errorf("close uploaded file %s: %w", name, err)
	}
	localHash := sha256.Sum256(data)
	remoteHash, err := c.fileSHA256(ctx, temporary)
	if err != nil {
		remote, readErr := c.readFile(ctx, temporary, false)
		if readErr != nil {
			return fmt.Errorf("verify %s: hash failed: %v; readback failed: %w", name, err, readErr)
		}
		if !bytes.Equal(remote, data) {
			return fmt.Errorf("verify %s: uploaded content differs", name)
		}
	} else if !bytes.Equal(remoteHash, localHash[:]) {
		return fmt.Errorf("verify %s: SHA-256 mismatch", name)
	}
	if _, err := c.Exec(ctx, activateUploadScript(name, temporary, backup)); err != nil {
		return fmt.Errorf("activate uploaded file %s: %w", name, err)
	}
	return nil
}

func createUploadScript(temporary, backup string) string {
	return fmt.Sprintf(`import os
for _p in (%s,%s):
 try: os.stat(_p)
 except OSError as _e:
  if _e.args[0] != 2: raise
 else: raise OSError('upload scratch path already exists: '+_p)
_f=open(%s,'wb')`, pythonString(temporary), pythonString(backup), pythonString(temporary))
}

func activateUploadScript(name, temporary, backup string) string {
	return fmt.Sprintf(`import os
_p=%s
_t=%s
_b=%s
_had=False
try:
 os.rename(_p,_b)
 _had=True
except OSError as _e:
 if _e.args[0] != 2: raise
try: os.rename(_t,_p)
except:
 if _had: os.rename(_b,_p)
 raise
if _had:
 try: os.remove(_b)
 except OSError: pass`, pythonString(name), pythonString(temporary), pythonString(backup))
}

func (c *Client) fileSHA256(ctx context.Context, name string) ([]byte, error) {
	code := fmt.Sprintf(`try: import hashlib as _hashlib
except ImportError: import uhashlib as _hashlib
import ubinascii
_h=_hashlib.sha256()
_f=open(%s,'rb')
while True:
 _d=_f.read(512)
 if not _d: break
 _h.update(_d)
_f.close()
print('%s'+ubinascii.hexlify(_h.digest()).decode())`, pythonString(name), hashMarker)
	output, err := c.Exec(ctx, code)
	if err != nil {
		return nil, err
	}
	encoded, err := markedLine(output, hashMarker)
	if err != nil {
		return nil, err
	}
	digest, err := hex.DecodeString(encoded)
	if err != nil || len(digest) != sha256.Size {
		return nil, fmt.Errorf("invalid SHA-256 response %q", encoded)
	}
	return digest, nil
}

// MkdirAll creates a device directory hierarchy and tolerates existing paths.
func (c *Client) MkdirAll(ctx context.Context, directory string) error {
	directory = path.Clean("/" + strings.TrimSpace(directory))
	if directory == "/" || directory == "." {
		return nil
	}
	code := fmt.Sprintf(`import os
_p=''
for _n in %s.strip('/').split('/'):
 _p += '/'+_n
 try: os.mkdir(_p)
 except OSError: pass`, pythonString(directory))
	if _, err := c.Exec(ctx, code); err != nil {
		return fmt.Errorf("create directory %s: %w", directory, err)
	}
	return nil
}

func pythonString(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func markedLine(output []byte, marker string) (string, error) {
	if lines := splitMarkedLines(output, marker); len(lines) > 0 {
		return lines[0], nil
	}
	return "", fmt.Errorf("response marker %s missing", marker)
}
