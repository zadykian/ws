package macos

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

// ending is a step that leaves Finish for the end of a run.
type ending struct {
	name string
	plan error
	done func(context.Context) error
}

func (e ending) Name() string    { return e.name }
func (e ending) WaitsOn() string { return "" }
func (e ending) Plan(context.Context) (step.Plan, error) {
	return step.Plan{}, e.plan
}
func (e ending) Apply(context.Context, step.Plan) error { return nil }
func (e ending) Finish(ctx context.Context) error       { return e.done(ctx) }

func TestRunFinishesNoStepLeftOut(t *testing.T) {
	finish := func(context.Context) error { return errors.New("a step left out finished") }
	steps := []step.Step{ending{name: "git", done: finish}}
	left := func(_, _ string) bool { return true }
	if _, err := run(t.Context(), steps, step.Options{LeftOut: left}); err != nil {
		t.Error(err)
	}
}

func TestRunFinishesNothingOfAFailedRun(t *testing.T) {
	finish := func(context.Context) error { t.Error("a failed run finished a step"); return nil }
	broke := errors.New("plan failed")
	steps := []step.Step{ending{name: "one", plan: broke, done: finish}}
	if _, err := run(t.Context(), steps, step.Options{}); !errors.Is(err, broke) {
		t.Errorf("err = %v; want the plan's", err)
	}
}

func TestRunFinishesEachStepThatRan(t *testing.T) {
	var done []string
	finish := func(name string) func(context.Context) error {
		return func(context.Context) error { done = append(done, name); return nil }
	}
	steps := []step.Step{ending{name: "one", done: finish("one")}, ending{name: "two", done: finish("two")}}
	if _, err := run(t.Context(), steps, step.Options{}); err != nil || len(done) != 2 {
		t.Errorf("finished %q, err = %v; want both", done, err)
	}
}

// seen is an observer that keeps the names of the steps whose events it took, in order.
type seen []string

func (l *seen) Observe(e step.Event) {
	if !slices.Contains(*l, e.Step) {
		*l = append(*l, e.Step)
	}
}

// reached runs m under o with every step left out, so that a plan which fails on the fake machine
// stops nothing, and returns the steps that took part.
func reached(t *testing.T, o Options) []string {
	var got seen
	m := Mac{M: effecttest.NewMachine(), Out: &effecttest.Out{}, Home: "/h"}
	o.Observer = &got
	o.LeftOut = func(_, _ string) bool { return true }
	if _, err := m.Run(t.Context(), o); err != nil {
		t.Fatalf("Run: %v", err)
	}
	return got
}

func TestMacRunEndsAnApplyWithSSHAgain(t *testing.T) {
	if got, want := reached(t, Options{}), append(Names(), "ssh again"); !slices.Equal(got, want) {
		t.Errorf("an apply ran %q; want %q", got, want)
	}
	if got := reached(t, Options{Options: step.Options{Check: true}}); !slices.Equal(got, Names()) {
		t.Errorf("a check ran %q; want %q", got, Names())
	}
}

func TestMacRunTakesOnlyTheStepsNamed(t *testing.T) {
	if got := reached(t, Options{Only: []string{"zed"}}); !slices.Equal(got, []string{"zed"}) {
		t.Errorf("ran %q; want zed alone, with no ssh again", got)
	}
	if got := reached(t, Options{Only: []string{"zed", "git"}}); !slices.Equal(got, []string{"git", "zed"}) {
		t.Errorf("ran %q; want git and zed, in the order of the steps", got)
	}
}

func TestMacRowsOfACheckHoldNoSSHAgain(t *testing.T) {
	names := func(steps []step.Step) []string {
		var got []string
		for _, s := range steps {
			got = append(got, s.Name())
		}
		return got
	}
	if got := names(Mac{}.Rows(Options{Options: step.Options{Check: true}})); !slices.Equal(got, Names()) {
		t.Errorf("a check's rows = %q; want %q", got, Names())
	}
	got := names(Mac{}.Rows(Options{}))
	if want := append(Names(), "ssh again"); !slices.Equal(got, want) {
		t.Errorf("an apply's rows = %q; want %q", got, want)
	}
}

// pending is a step with one change to make, which a second plan finds made.
type pending struct {
	name, waits string
	made        bool
}

func (p *pending) Name() string    { return p.name }
func (p *pending) WaitsOn() string { return p.waits }
func (p *pending) Plan(context.Context) (step.Plan, error) {
	if p.made {
		return step.Plan{}, nil
	}
	return step.Plan{Changes: []step.Change{pendingChange("x")}}, nil
}
func (p *pending) Apply(context.Context, step.Plan) error { p.made = true; return nil }

type pendingChange string

func (c pendingChange) String() string { return string(c) }

func TestRunAsksLeftOutOnlyOfAStepWithARow(t *testing.T) {
	keepSSH := func(_, name string) bool { return name != "ssh" }
	ssh, again := &pending{name: "ssh"}, &pending{name: "ssh again", waits: "ssh"}
	if _, err := run(t.Context(), []step.Step{ssh, again}, step.Options{LeftOut: keepSSH}); err != nil {
		t.Fatal(err)
	}
	if !ssh.made || !again.made {
		t.Errorf("ssh made = %v, ssh again made = %v; want both, as the choice keeps ssh", ssh.made, again.made)
	}
	var got []step.Event
	obs := observed(func(e step.Event) { got = append(got, e) })
	ssh, again = &pending{name: "ssh"}, &pending{name: "ssh again", waits: "ssh"}
	left := func(_, _ string) bool { return true }
	if _, err := run(t.Context(), []step.Step{ssh, again}, step.Options{LeftOut: left, Observer: obs}); err != nil {
		t.Fatal(err)
	}
	if again.made || ssh.made {
		t.Error("a step ran where the choice left ssh out")
	}
	last := got[len(got)-1]
	if last.Step != "ssh again" || last.Kind != step.Skipped || last.After != "ssh" {
		t.Errorf("last event = %+v; want ssh again skipped behind ssh", last)
	}
}

// observed is an observer made of a function.
type observed func(step.Event)

func (f observed) Observe(e step.Event) { f(e) }

func TestRunReturnsAStepsErrorAsItIs(t *testing.T) {
	for _, parent := range []string{"", "tools"} {
		o := step.Options{Parent: parent}
		steps := func() []step.Step {
			return []step.Step{&pending{name: "git"}, ending{name: "zed", plan: errors.New("boom")}}
		}
		_, want := step.RunWith(t.Context(), steps(), o)
		_, got := run(t.Context(), steps(), o)
		if want == nil || got == nil || got.Error() != want.Error() {
			t.Errorf("parent %q: error = %v, want step.RunWith's %v", parent, got, want)
		}
	}
}
