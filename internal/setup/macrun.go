package setup

import (
	"context"
	"io"
	"slices"
	"time"

	"github.com/zadykian/ws/step"
)

// Options says how Run runs.
type Options struct {
	step.Options
	// Only names the one step that runs, alone. Empty runs them all.
	Only string
}

// Run runs the steps that o names, or every step. A run that applies and names no step ends with
// ssh again, as cld's restore tabs may have changed ssh's file. Then each step of a run that passed
// finishes, but one left out. It reports what step.RunWith does.
func (m Mac) Run(ctx context.Context, o Options) (bool, error) {
	return run(ctx, m.ofRun(o), o.Options)
}

// Rows returns the steps of the run that o makes, for their names and waits alone: they hold no
// effects, so a run of them fails at once.
func (m Mac) Rows(o Options) []step.Step {
	return Mac{Text: io.Discard}.ofRun(o)
}

// ofRun returns the steps of the run that o makes.
func (m Mac) ofRun(o Options) []step.Step {
	steps := m.steps(o.Check)
	switch {
	case o.Only != "":
		return slices.DeleteFunc(steps, func(s step.Step) bool { return s.Name() != o.Only })
	case o.Check:
		// Nothing is applied, so a second plan would only repeat the first.
		return steps
	}
	return append(steps, m.ssh(true))
}

// ssh returns the step for ssh's file, or for its second run.
func (m Mac) ssh(again bool) step.Step {
	return SSH{Conf: m.Conf, Home: m.Home, M: m.M, Out: m.Out, Now: time.Now, Again: again}
}

// finisher is a step that leaves something for the end of a run that passed.
type finisher interface{ finish(context.Context) error }

// run runs steps under o, then what each leaves for the end of a run that passed. A step left out
// leaves nothing, as it applied nothing. o.LeftOut is asked of the steps with a row alone: ssh
// again has none, as a check has no second plan, so it follows ssh by its wait. It reports what
// step.RunWith does.
func run(ctx context.Context, steps []step.Step, o step.Options) (bool, error) {
	if left := o.LeftOut; left != nil {
		rows := Names()
		o.LeftOut = func(parent, name string) bool { return slices.Contains(rows, name) && left(parent, name) }
	}
	changed, err := step.RunWith(ctx, steps, o)
	for _, s := range steps {
		left := o.LeftOut != nil && o.LeftOut(o.Parent, s.Name())
		if f, ok := s.(finisher); ok && err == nil && !left {
			err = f.finish(ctx)
		}
	}
	return changed, err
}
