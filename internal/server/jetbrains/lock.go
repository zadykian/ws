package jetbrains

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// withLock holds flock(2)'s lock on the directory while fn runs, as flock(1) does for a path. A
// run unpacks or removes a build only under it, so that the weekly update and a run by hand never
// unpack one build at once, nor remove the tree the other unpacks. It waits for the lock.
func withLock(dir string, fn func() error) error {
	f, err := os.Open(dir) //nolint:gosec // the directory is Gateway's, from the user's home
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }() // a read only descriptor; closing it drops the lock
	for {
		err = unix.Flock(int(f.Fd()), unix.LOCK_EX)
		if !errors.Is(err, unix.EINTR) {
			break
		}
	}
	if err != nil {
		return err
	}
	return fn()
}
