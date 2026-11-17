package engine

import (
	"io"

	"github.com/zadykian/ws/engine/core/effect"
)

// Call is a verb's call as the facade reads it. engine/cli fills it from cobra and the Env, so that
// the facade names neither.
type Call struct {
	// Path is the command's path, which lines name.
	Path string
	// Out is stdout, and Err is stderr.
	Out, Err io.Writer
	// JSON is --json, which prints data in place of lines.
	JSON bool
	// Check is --check, and Selected tells that --only was given, even with no name.
	Check, Selected bool
	Machine         effect.Machine
	Getenv          func(string) string
}
