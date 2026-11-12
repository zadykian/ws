package setup

import (
	"context"
	"errors"
	"testing"

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
