//go:build linux

package base

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

// osDisk is the disk of the machine that ws runs on.
type osDisk struct{}

// Statfs returns the MiB that a non-root writer can use, as df's avail column does but not rounded
// up.
func (osDisk) Statfs(path string) (int64, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return 0, fmt.Errorf("statfs %s: %w", path, err)
	}
	return int64(st.Bavail) * int64(st.Bsize) / mib, nil //nolint:gosec // block counts of a disk fit int64
}

// Stat describes the path.
func (osDisk) Stat(path string) (swapFileInfo, error) {
	fi, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return swapFileInfo{}, nil
	}
	if err != nil {
		return swapFileInfo{}, err
	}
	info := swapFileInfo{Exists: true, Size: fi.Size(), Mode: fi.Mode()}
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		info.UID, info.GID = int(st.Uid), int(st.Gid)
	}
	return info, nil
}

// Restrict makes the file root's, mode 0600.
func (osDisk) Restrict(path string) error {
	if err := os.Chown(path, 0, 0); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

// Rename renames the file.
func (osDisk) Rename(oldPath, newPath string) error { return os.Rename(oldPath, newPath) }

// Remove removes the file.
func (osDisk) Remove(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
