//go:build darwin

package cli

import "golang.org/x/sys/unix"

// isTerminal tells whether the file descriptor is a terminal.
func isTerminal(fd int) bool {
	_, err := unix.IoctlGetTermios(fd, unix.TIOCGETA)
	return err == nil
}
