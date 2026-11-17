// Package kind holds ws's kinds: resource types written once, which a role's spec names as units,
// KIND/NAME. A unit is a step.Step built on the resource packages, so that a run of units plans,
// applies and plans again as a role's steps do, and no role's word sits in a kind.
package kind

import "github.com/zadykian/ws/engine/core/effect"

// Env is what a kind builds its units on: the machine, systemd's units, the lines, and the side's
// values, which fill the specs and their templates.
type Env struct {
	M    effect.Machine
	U    effect.Units
	Out  effect.Out
	Data any
}
