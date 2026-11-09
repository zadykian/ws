package tui

import (
	"context"

	"github.com/zadykian/ws/effect"
)

// Effects is what a work beside a screen runs on, besides the Bridge, which takes its output and
// its questions.
type Effects struct {
	// M is the machine as the screen takes it.
	M effect.Machine
	// Talks runs a command that talks to the maintainer: behind the screen's suspension on M, or a
	// fake's own. A fake with none leaves it nil.
	Talks effect.Foreground
	// Stop is the run's step.Options.Stop.
	Stop func() error
}

// Effects returns the screen's effects, which Run made of Config.Machine as the screen began. A
// nil Bridge, a test's, has none.
func (b *Bridge) Effects() Effects {
	if b == nil {
		return Effects{}
	}
	return b.effects
}

// effectsOn returns m's effects under the screen, for the works that ctx runs. An OSMachine is
// wrapped by Machine, once for the screen: its works share it, as each wrapping would hold sudo's
// ticket until the screen ends. A fake keeps its own commands, as it has no terminal to keep them
// off.
func (b *Bridge) effectsOn(ctx context.Context, m effect.Machine) Effects {
	if machine, ok := m.(effect.OSMachine); ok {
		m = b.Machine(ctx, machine)
	}
	talks, _ := m.(effect.Foreground)
	if machine, ok := m.(effect.OSMachine); ok {
		talks = b.Foreground(machine)
	}
	return Effects{M: m, Talks: talks, Stop: b.Stop(ctx)}
}
