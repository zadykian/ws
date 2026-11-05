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

// Run plans each step in order. Under check it prints each plan and applies nothing, and skips a
// step whose awaited step has changes to make. Otherwise it applies each plan, plans again, and
// fails where the second plan is not empty. It reports whether any step had changes to make.
func Run(ctx context.Context, steps []Step, out effect.Out, check bool) (changed bool, err error) {
	pending := map[string]bool{} // the earlier steps, whether each has changes to make
	for _, s := range steps {
		did, err := settle(ctx, s, out, check, pending)
		changed = changed || did
		if err != nil {
			return changed, err
		}
	}
	return changed, nil
}

// settle takes one step as far as the run goes, and reports whether it had changes to make.
func settle(ctx context.Context, s Step, out effect.Out, check bool, pending map[string]bool) (changed bool, err error) {
	name := s.Name()
	waitsOn := s.WaitsOn()
	if waitsOn != "" {
		if _, ok := pending[waitsOn]; !ok {
			return false, fmt.Errorf("%s waits on %q, which is not an earlier step", name, waitsOn)
		}
		if check && pending[waitsOn] {
			out.Say("%s: after %s", name, waitsOn)
			pending[name] = true
			return true, nil
		}
	}
	p, err := s.Plan(ctx)
	if err != nil {
		return false, fmt.Errorf("%s: planning: %w", name, err)
	}
	pending[name] = len(p.Changes) != 0
	if !pending[name] {
		return false, nil
	}
	if check {
		report(out, name, p)
		return true, nil
	}
	out.Say("%s: applying %d change(s)", name, len(p.Changes))
	if err := s.Apply(ctx, p); err != nil {
		return true, fmt.Errorf("%s: applying: %w", name, err)
	}
	again, err := s.Plan(ctx)
	if err != nil {
		return true, fmt.Errorf("%s: planning again: %w", name, err)
	}
	if len(again.Changes) != 0 {
		return true, fmt.Errorf("%s: still has %d change(s) after applying, the first: %s",
			name, len(again.Changes), again.Changes[0])
	}
	pending[name] = false
	return true, nil
}

func report(out effect.Out, name string, p Plan) {
	for _, c := range p.Changes {
		out.Say("%s: %s", name, c)
	}
}
