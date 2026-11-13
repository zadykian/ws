package engine

import (
	"io"

	"github.com/zadykian/ws/engine/core/effect"
)

// Call is a verb's call as the facade reads it. engine/cli fills it from cobra and the Env, so that
// the facade names neither.
type Call struct {
	// Path is the command's path, which titles, logs and lines name.
	Path string
	In   io.Reader
	// Out is stdout, and Err is stderr.
	Out, Err io.Writer
	// Plain and JSON are the flags that turn the screen off.
	Plain, JSON bool
	// Check is --check, and Selected tells that --only was given, even with no name.
	Check, Selected bool
	Machine         effect.Machine
	Getenv          func(string) string
	// TTY, ErrTTY and OutTTY tell whether stdin, stderr and stdout are terminals.
	TTY, ErrTTY, OutTTY bool
	// LockDir holds the wizards' lock.
	LockDir string
}
