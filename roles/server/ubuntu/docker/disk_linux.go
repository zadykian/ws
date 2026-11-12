//go:build linux

package docker

import "golang.org/x/sys/unix"

// StatfsSize is the DiskSize of a Linux machine: the filesystem's blocks, each of the fragment
// size, as df --block-size=1 counts them.
func StatfsSize(path string) (uint64, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return 0, err
	}
	return st.Blocks * uint64(st.Frsize), nil //nolint:gosec // Frsize is a block size, never negative
}
