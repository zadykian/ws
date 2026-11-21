// Package effecttest holds the fakes of package effect, for the tests of steps.
package effecttest

import (
	"fmt"

	"github.com/zadykian/ws/engine/core/effect"
)

// Out records what is said, one entry per call, as "say: ...", "raw: ...", "warn: ...", "diff: NAME" or
// "by hand: ...".
type Out struct {
	Lines []string
}

// Say records a line.
func (o *Out) Say(format string, a ...any) { o.add("say: ", format, a) }

// Raw records a line that has no prefix.
func (o *Out) Raw(format string, a ...any) { o.add("raw: ", format, a) }

// Warn records a warning.
func (o *Out) Warn(format string, a ...any) { o.add("warn: ", format, a) }

// Diff records the name of the diff and its text.
func (o *Out) Diff(name string, before, after []byte) {
	o.Lines = append(o.Lines, "diff: "+name+"\n"+effect.Diff(name, before, after))
}

// ByHand records a line to do by hand.
func (o *Out) ByHand(format string, a ...any) { o.add("by hand: ", format, a) }

func (o *Out) add(prefix, format string, a []any) {
	o.Lines = append(o.Lines, prefix+fmt.Sprintf(format, a...))
}
