// Package step holds the Step interface, which each step of a machine's setup implements, and
// the runner that plans and applies the steps in order.
package step

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/effect"
)

// Change is one change a step would make, as --check prints it.
type Change interface{ String() string }

// Plan is what a step finds it has to do.
type Plan struct {
	// Changes is empty once the step is done.
	Changes []Change
}

// Step sets up one part of a machine's setup. A step holds its effects as fields and reads no other step's state.
type Step interface {
	Name() string
	// WaitsOn names an earlier step that must be done first, or is empty. It is known before Plan,
	// as a plan may fail where that step's changes are still to come.
	WaitsOn() string
	// Plan reads alone and changes nothing.
	Plan(ctx context.Context) (Plan, error)
	// Apply applies the changes of a plan that the step made.
	Apply(ctx context.Context, p Plan) error
}

// Options says how RunWith runs.
type Options struct {
	// Observer takes the run's events. Nil takes none, and the run prints nothing.
	Observer Observer
	// Check plans each step and applies none.
	Check bool
	// Parent names the role or stage of a nested run. A run of its own leaves it empty.
	Parent string
	// LeftOut says which steps the run plans and doesn't apply, by the run's Parent and the step's
	// name. Nil leaves none out. A step is planned all the same, as later steps read its state.
	// Check leaves them out too, as it shows what the same run would make. A plan that fails here
	// stops nothing, as the step wasn't asked for, and what waits on the step counts it as pending.
	LeftOut func(parent, step string) bool
	// Stop is read before each step, and an error ends the run there, with the steps left Skipped.
	// Nil never stops, as a run in lines ends mid-step or not at all.
	Stop func() error
}

// Run is RunWith with a Printer on out, for a caller that follows a run by its lines.
func Run(ctx context.Context, steps []Step, out effect.Out, check bool) (changed bool, err error) {
	return RunWith(ctx, steps, Options{Observer: Printer{Out: out}, Check: check})
}

// RunWith plans each step in order. Under check it applies nothing, and skips a step whose awaited
// step has changes to make. A step left out is planned and not applied, and the steps that wait on
// it, if it has changes, are skipped as under check. Otherwise it applies each plan, plans again,
// and fails where the second plan is not empty. It stops at the first failure, or where o.Stop
// says, and the steps past it end Skipped. A left-out step's failed plan is no failure, unless ctx
// ended it. It reports whether any step it didn't leave out had changes to make.
func RunWith(ctx context.Context, steps []Step, o Options) (changed bool, err error) {
	pending := map[string]bool{} // the earlier steps, whether each has changes to make
	for i, s := range steps {
		if err := o.stopped(); err != nil {
			o.unreached(steps, i)
			return changed, err
		}
		e := o.settle(ctx, s, o.event(i, s), pending)
		o.observe(e)
		// A step skipped needs no count: the step it waits on counts, or is left out.
		changed = changed || !e.LeftOut && len(e.Changes) != 0
		if e.Kind == Failed {
			o.unreached(steps, i+1)
			return changed, e.Err
		}
	}
	return changed, nil
}

// stopped is why the run goes no further, or nil where it goes on.
func (o Options) stopped() error {
	if o.Stop == nil {
		return nil
	}
	return o.Stop()
}

// settle takes one step as far as the run goes, and returns its last event for the run to send.
func (o Options) settle(ctx context.Context, s Step, e Event, pending map[string]bool) Event {
	name := e.Step
	e.LeftOut = o.LeftOut != nil && o.LeftOut(o.Parent, name)
	waitsOn := s.WaitsOn()
	if waitsOn != "" {
		if _, ok := pending[waitsOn]; !ok {
			return e.failed(fmt.Errorf("%s waits on %q, which is not an earlier step", name, waitsOn))
		}
		if (o.Check || o.LeftOut != nil) && pending[waitsOn] {
			pending[name] = true
			e.After = waitsOn
			return e.as(Skipped)
		}
	}
	p, err := s.Plan(ctx)
	if err != nil {
		e = e.failed(fmt.Errorf("%s: planning: %w", name, err))
		if e.LeftOut && ctx.Err() == nil {
			pending[name] = true
			return e.as(Skipped)
		}
		return e
	}
	e.Changes = p.Changes
	o.observe(e.as(Planned))
	pending[name] = len(p.Changes) != 0
	if o.Check || e.LeftOut || !pending[name] {
		return e.as(Done)
	}
	o.observe(e.as(Started))
	if err := s.Apply(ctx, p); err != nil {
		return e.failed(fmt.Errorf("%s: applying: %w", name, err))
	}
	again, err := s.Plan(ctx)
	if err != nil {
		return e.failed(fmt.Errorf("%s: planning again: %w", name, err))
	}
	if len(again.Changes) != 0 {
		return e.failed(fmt.Errorf("%s: still has %d change(s) after applying, the first: %s",
			name, len(again.Changes), Headline(again.Changes[0])))
	}
	pending[name] = false
	return e.as(Done)
}

// Unreached ends each step as Skipped, for the nested runs that a failed run before them never
// starts: each step of a run gets its last event, and a screen has no other word of these.
func Unreached(steps []Step, o Options) { o.unreached(steps, 0) }

func (o Options) unreached(steps []Step, from int) {
	for i := from; i < len(steps); i++ {
		o.observe(o.event(i, steps[i]).as(Skipped))
	}
}

func (o Options) event(i int, s Step) Event {
	return Event{Parent: o.Parent, Step: s.Name(), Index: i, Check: o.Check}
}

func (o Options) observe(e Event) {
	if o.Observer != nil {
		o.Observer.Observe(e)
	}
}
