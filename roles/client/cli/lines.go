package cli

import (
	"github.com/zadykian/ws/engine"
	"github.com/zadykian/ws/engine/core/effect"
)

// linesOf returns the lines that the command at path prints on out.
func linesOf(out effect.Out, path string) engine.Lines {
	return engine.Lines{Out: out, Prog: path, Seen: map[string]bool{}}
}
