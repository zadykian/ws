package apply

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/server/roles"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

type change string

func (c change) String() string { return string(c) }

// todo is a step with one change until it is applied.
type todo struct{ done *bool }

func (todo) Name() string    { return "todo" }
func (todo) WaitsOn() string { return "" }
func (t todo) Plan(context.Context) (step.Plan, error) {
	if *t.done {
		return step.Plan{}, nil
	}
	return step.Plan{Changes: []step.Change{change("x")}}, nil
}
func (t todo) Apply(context.Context, step.Plan) error { *t.done = true; return nil }

func TestRunSteps(t *testing.T) {
	var done bool
	ported := func(values.Values, effect.Machine, effect.Out) []step.Step { return []step.Step{todo{&done}} }
	segs, err := Segments([]roles.Role{{Name: "c", Steps: ported}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, check := range []bool{false, true} {
		done = false
		c := Config{Options: Options{Check: check}, M: effecttest.NewMachine(), Out: &effecttest.Out{}}
		got, err := Run(context.Background(), c, segs)
		if err != nil || !slices.Equal(got, []string{"c"}) || done == check {
			t.Errorf("check %v: Run = %q, %v, applied %v", check, got, err, done)
		}
	}
}

func TestRunRefusesAPlaybookSegment(t *testing.T) {
	segs, _ := Segments([]roles.Role{{Name: "a"}}, nil)
	c := Config{M: effecttest.NewMachine(), Out: &effecttest.Out{}}
	if _, err := Run(context.Background(), c, segs); err == nil || !strings.Contains(err.Error(), "not run yet") {
		t.Errorf("err = %v, want not run yet", err)
	}
}
