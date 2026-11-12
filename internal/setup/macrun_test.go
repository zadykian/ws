package setup

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/step"
)

// ending is a step that leaves finish for the end of a run.
type ending struct {
	name   string
	plan   error
	finish func(context.Context) error
}

func (e ending) Name() string    { return e.name }
func (e ending) WaitsOn() string { return "" }
func (e ending) Plan(context.Context) (step.Plan, error) {
	return step.Plan{}, e.plan
}
func (e ending) Apply(context.Context, step.Plan) error { return nil }
func (e ending) Finish(ctx context.Context) error       { return e.finish(ctx) }

func TestRunFinishesNoStepLeftOut(t *testing.T) {
	finish := func(context.Context) error { return errors.New("a step left out finished") }
	steps := []step.Step{ending{name: "one", finish: finish}}
	left := func(_, _ string) bool { return true }
	if _, err := run(t.Context(), steps, step.Options{LeftOut: left}); err != nil {
		t.Error(err)
	}
}

func TestRunFinishesNothingOfAFailedRun(t *testing.T) {
	finish := func(context.Context) error { t.Error("a failed run finished a step"); return nil }
	broke := errors.New("plan failed")
	steps := []step.Step{ending{name: "one", plan: broke, finish: finish}}
	if _, err := run(t.Context(), steps, step.Options{}); !errors.Is(err, broke) {
		t.Errorf("err = %v; want the plan's", err)
	}
}

func TestRunFinishesEachStepThatRan(t *testing.T) {
	var done []string
	finish := func(name string) func(context.Context) error {
		return func(context.Context) error { done = append(done, name); return nil }
	}
	steps := []step.Step{ending{name: "one", finish: finish("one")}, ending{name: "two", finish: finish("two")}}
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

func TestMacRunTakesOnlyTheStepNamed(t *testing.T) {
	if got := reached(t, Options{Only: "zed"}); !slices.Equal(got, []string{"zed"}) {
		t.Errorf("ran %q; want zed alone, with no ssh again", got)
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
