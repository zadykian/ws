//go:build linux

package swap

import (
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

// Allocate reserves the blocks with fallocate, which writes nothing. Where the file system does
// not support it, it writes zeros, and on no other error: where the disk lacks room, zeros would
// fill it.
func (osDisk) Allocate(path string, size int64) (err error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) //nolint:gosec // the step names the path
	if err != nil {
		return err
	}
	defer func() {
		if err = errors.Join(err, f.Close()); err != nil {
			err = errors.Join(err, os.Remove(path))
		}
	}()
	if err := f.Chmod(0o600); err != nil {
		return err
	}
	err = unix.Fallocate(int(f.Fd()), 0, 0, size*mib)
	if errors.Is(err, unix.EOPNOTSUPP) {
		_, err = io.CopyN(f, zeros{}, size*mib)
	}
	if err != nil {
		return fmt.Errorf("allocate %s: %w", path, err)
	}
	return f.Sync()
}

// zeros reads zero bytes without end.
type zeros struct{}

func (zeros) Read(p []byte) (int, error) { clear(p); return len(p), nil }
