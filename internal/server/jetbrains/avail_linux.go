//go:build linux

package jetbrains

import "golang.org/x/sys/unix"

// StatfsAvail is the Avail of a Linux machine: the bytes free to a user, as df --output=avail
// --block-size=1 counts them.
func StatfsAvail(path string) (uint64, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return 0, err
	}
	return st.Bavail * uint64(st.Frsize), nil //nolint:gosec // Frsize is a block size, never negative
}
