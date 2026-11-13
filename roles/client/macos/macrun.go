package macos

import (
	"context"
	"errors"
	"io"
	"slices"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/setup"
)

// Options says how Run runs.
type Options struct {
	step.Options
	// Only names the steps that run, alone. Empty runs them all.
	Only []string
}

// Run runs the steps that o names, or every step. A run that applies and names no step ends with
// ssh again, as cld's restore tabs may have changed ssh's file. Then each step of a run that passed
// finishes, but one left out. It reports whether any step changed, and false past an error.
func (m Mac) Run(ctx context.Context, o Options) (bool, error) {
	return run(ctx, m.ofRun(o), o.Options)
}

// Rows returns the steps of the run that o makes, for their names and waits alone: they hold no
// effects, so a run of them fails at once.
func (m Mac) Rows(o Options) []step.Step {
	return Mac{Text: io.Discard}.ofRun(o)
}

// ofRun returns the steps of the run that o makes: those of the roles it names, in order.
func (m Mac) ofRun(o Options) []step.Step {
	roles := m.Roles(o.Check)
	if len(o.Only) > 0 {
		roles = slices.DeleteFunc(roles, func(r role.Role) bool { return !slices.Contains(o.Only, r.Name) })
	}
	var steps []step.Step
	for _, r := range roles {
		steps = append(steps, r.Steps(m.M, nil, m.Out)...)
	}
	if len(o.Only) == 0 && !o.Check {
		// Nothing is applied under a check, so a second plan would only repeat the first.
		steps = append(steps, m.ssh(true))
	}
	return steps
}

// ssh returns the step for ssh's file, or for its second run.
func (m Mac) ssh(again bool) step.Step {
	return setup.SSH{Conf: m.Conf, Home: m.Home, M: m.M, Out: m.Out, Now: time.Now, Again: again}
}

// finisher is a step that leaves something for the end of a run that passed.
type finisher interface{ Finish(context.Context) error }

// run runs steps under o as one role, named by o.Parent, through role.Run, then what each leaves
// for the end of a run that passed. One role keeps the rows, the waits and the events of the steps
// as step.RunWith gives them. A step left out leaves nothing, as it applied nothing. o.LeftOut is
// asked of the steps with a row alone: ssh again has none, as a check has no second plan, so it
// follows ssh by its wait. It reports whether any step changed, and false past an error.
func run(ctx context.Context, steps []step.Step, o step.Options) (bool, error) {
	if left := o.LeftOut; left != nil {
		rows := Names()
		o.LeftOut = func(parent, name string) bool { return slices.Contains(rows, name) && left(parent, name) }
	}
	one := role.Role{Name: o.Parent, Steps: func(effect.Machine, effect.Units, effect.Out) []step.Step {
		return steps
	}}
	c := role.Config{Options: role.Options{Check: o.Check}, Observer: o.Observer, Stop: o.Stop, LeftOut: o.LeftOut}
	changed, err := role.Run(ctx, c, []role.Role{one})
	if err != nil {
		// role.Run puts the role's name before the step's error, and callers read the error bare.
		err = errors.Unwrap(err)
	}
	for _, s := range steps {
		left := o.LeftOut != nil && o.LeftOut(o.Parent, s.Name())
		if f, ok := s.(finisher); ok && err == nil && !left {
			err = f.Finish(ctx)
		}
	}
	return len(changed) > 0, err
}
