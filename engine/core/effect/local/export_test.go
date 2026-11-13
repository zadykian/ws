package local

import (
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

// RunAs has ws run as uid, which may write no directory where writable is false, for the test's
// length.
func RunAs(t *testing.T, uid int, writable bool) {
	oldUID, oldAccess := euid, access
	euid, access = func() int { return uid }, unix.Access
	if !writable {
		access = func(string, uint32) error { return syscall.EACCES }
	}
	t.Cleanup(func() { euid, access = oldUID, oldAccess })
}
