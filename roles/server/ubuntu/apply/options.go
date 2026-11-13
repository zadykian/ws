// Package apply runs the server's roles for ws server ubuntu apply, each as Go steps.
package apply

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// Role is a name and the steps it makes on a machine, once it holds the values it reads.
type Role struct {
	// Name is the role's name, as --only takes it.
	Name string
	// Steps returns the role's steps, in the order they run.
	Steps func(m effect.Machine, u effect.Units, out effect.Out) []step.Step
}

// Options are the flags of ws server ubuntu apply.
type Options struct {
	// Check plans the steps and applies none.
	Check bool
	// Only names what runs, each by ROLE or ROLE/STEP, or is empty for all. The other steps of a
	// role named by ROLE/STEP alone are planned and left out, as later steps read their state.
	Only []string
}
