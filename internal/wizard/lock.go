// Package wizard holds ws mac wizard: the lock and the sleep guard of a run, the closing list of what
// stays by hand, and the stages, each a step.Step.
package wizard

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/effect"
)

// LockName is the lock's name, a directory in the temporary directory that holds a run's own files.
const LockName = "ws-wizard.lock"

// Lock takes a run's lock, a directory in tmp, or fails where another run holds it. The returned
// release removes the lock, and the files in it. A run calls it on every way out.
func Lock(ctx context.Context, m effect.Machine, tmp string) (path string, release func(), err error) {
	path = strings.TrimRight(tmp, "/") + "/" + LockName
	if err := m.Mkdir(path, 0o700); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return "", nil, fmt.Errorf("another run holds %s: wait for it to end, or remove it where none runs", path)
		}
		return "", nil, fmt.Errorf("cannot make %s", path)
	}
	return path, func() {
		// The context of a run that a signal ended is done, and the lock must go all the same.
		_, _ = m.Run(context.WithoutCancel(ctx), "rm", "-rf", path)
	}, nil
}
