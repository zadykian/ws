//go:build !linux && !darwin

package cli

// isTerminal is false where ws runs no command: no target is a terminal there.
func isTerminal(int) bool { return false }
