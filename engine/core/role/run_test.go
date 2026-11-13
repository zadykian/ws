package role

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

type change string

func (c change) String() string { return string(c) }

// todo is a step with one change until applied.
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

func TestRun(t *testing.T) {
	for _, check := range []bool{false, true} {
		var a, b bool
		c := Config{Options: Options{Check: check}, M: effecttest.NewMachine(), Out: &effecttest.Out{}}
		got, err := Run(context.Background(), c, []Role{role("a", todo{&a}), role("b", todo{&b})})
		if err != nil || !slices.Equal(got, []string{"a", "b"}) || a == check || b == check {
			t.Errorf("check %v: changed = %q, a = %v, b = %v, err = %v", check, got, a, b, err)
		}
	}
}

// named is todo under a name of its own.
type named struct {
	todo
	name string
}

func (n named) Name() string { return n.name }

func TestRunTakesStepsByName(t *testing.T) {
	tests := []struct {
		name    string
		check   bool
		only    []string
		want    []string
		applied []string
	}{
		{"a step of a role", false, []string{"a/x"}, []string{"a"}, []string{"prepare", "x"}},
		{"a step beside a role", false, []string{"a/x", "b"}, []string{"a", "b"}, []string{"prepare", "x", "z"}},
		{"a step and its role", false, []string{"a/x", "a"}, []string{"a"}, []string{"prepare", "x", "y"}},
		{"a check", true, []string{"a/x"}, []string{"a"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			done := map[string]*bool{"prepare": new(bool), "x": new(bool), "y": new(bool), "z": new(bool)}
			a := role("a", named{todo{done["x"]}, "x"}, named{todo{done["y"]}, "y"})
			b := role("b", named{todo{done["z"]}, "z"})
			c := Config{Options: Options{Check: tt.check, Only: tt.only}, M: effecttest.NewMachine(),
				Out: &effecttest.Out{}, Prepare: []step.Step{named{todo{done["prepare"]}, "prepare"}}}
			list, err := Select([]Role{a, b}, tt.only)
			if err != nil {
				t.Fatal(err)
			}
			got, err := Run(context.Background(), c, list)
			if err != nil || !slices.Equal(got, tt.want) {
				t.Errorf("changed = %q, err = %v, want %q", got, err, tt.want)
			}
			var applied []string
			for _, name := range []string{"prepare", "x", "y", "z"} {
				if *done[name] {
					applied = append(applied, name)
				}
			}
			if !slices.Equal(applied, tt.applied) {
				t.Errorf("applied %q, want %q", applied, tt.applied)
			}
		})
	}
}

// A role whose named step has nothing to do changed nothing, whatever the steps beside it hold.
func TestRunNamesNoRoleForStepsLeftOut(t *testing.T) {
	for _, check := range []bool{false, true} {
		done, pending := true, false
		a := role("a", named{todo{&done}, "x"}, named{todo{&pending}, "y"})
		c := Config{Options: Options{Check: check, Only: []string{"a/x"}}, M: effecttest.NewMachine(), Out: &effecttest.Out{}}
		got, err := Run(context.Background(), c, []Role{a})
		if err != nil || len(got) != 0 || pending {
			t.Errorf("check %v: changed = %q, y applied = %v, err = %v", check, got, pending, err)
		}
	}
}

// failing is a step whose plan fails.
type failing struct{ todo }

func (failing) Plan(context.Context) (step.Plan, error) { return step.Plan{}, errors.New("boom") }

// lasts keeps the last event of each step, as parent/step and its kind.
type lasts []string

func (l *lasts) Observe(e step.Event) {
	kinds := map[step.Kind]string{step.Done: "done", step.Failed: "failed", step.Skipped: "skipped"}
	if kind, ok := kinds[e.Kind]; ok {
		*l = append(*l, e.Parent+"/"+e.Step+" "+kind)
	}
}

// role is a role of the steps list.
func role(name string, list ...step.Step) Role {
	return Role{Name: name, Steps: func(effect.Machine, effect.Units, effect.Out) []step.Step {
		return list
	}}
}

func TestRunNamesEachRoleAndEndsThoseAFailureStopsShortOf(t *testing.T) {
	var done bool
	ok, bad := todo{&done}, failing{todo{&done}}
	tests := []struct {
		name    string
		prepare step.Step
		list    []Role
		want    lasts
	}{
		{"a role fails", ok, []Role{role("a", bad), role("b", ok)},
			lasts{"prepare/todo done", "a/todo failed", "b/todo skipped"}},
		{"prepare fails", bad, []Role{role("a", ok)}, lasts{"prepare/todo failed", "a/todo skipped"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			done = true
			var got lasts
			c := Config{M: effecttest.NewMachine(), Out: &effecttest.Out{}, Observer: &got, Prepare: []step.Step{tt.prepare}}
			if _, err := Run(context.Background(), c, tt.list); err == nil {
				t.Fatal("the run succeeded")
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("last events = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRunStopsBetweenTheRolesAndEndsThoseItStopsShortOf(t *testing.T) {
	stopped := errors.New("stopped")
	var a, b bool
	var got lasts
	c := Config{M: effecttest.NewMachine(), Out: &effecttest.Out{}, Observer: &got, Stop: func() error {
		if a {
			return stopped
		}
		return nil
	}}
	changed, err := Run(context.Background(), c, []Role{role("a", todo{&a}), role("b", todo{&b}), role("c", todo{&b})})
	if !errors.Is(err, stopped) || !slices.Equal(changed, []string{"a"}) || b {
		t.Errorf("changed = %q, err = %v, b = %v", changed, err, b)
	}
	if want := (lasts{"a/todo done", "b/todo skipped", "c/todo skipped"}); !slices.Equal(got, want) {
		t.Errorf("last events = %q, want %q", got, want)
	}
}

func TestSummary(t *testing.T) {
	for _, tc := range []struct {
		changed []string
		check   bool
		want    string
	}{
		{nil, false, "changed: none"},
		{[]string{"base", "fans"}, false, "changed: base, fans"},
		{[]string{"base"}, true, "would change: base"},
		{nil, true, "would change: none"},
	} {
		if got := Summary(tc.changed, tc.check); got != tc.want {
			t.Errorf("Summary(%q, %v) = %q, want %q", tc.changed, tc.check, got, tc.want)
		}
	}
}

// reports is a step that reports one item.
type reports struct{ todo }

func (reports) Report(context.Context) ([]step.Item, error) {
	return []step.Item{{Kind: step.PackageItem, Name: "git", Want: "installed", Found: "missing"}}, nil
}

func TestStateReadsPrepareAndEachRoleAndAppliesNone(t *testing.T) {
	var a, b bool
	role := func(name string, s step.Step) Role {
		return Role{Name: name, Steps: func(effect.Machine, effect.Units, effect.Out) []step.Step {
			return []step.Step{s}
		}}
	}
	c := Config{M: effecttest.NewMachine(), Out: &effecttest.Out{}, Prepare: []step.Step{todo{&a}}}
	got, err := State(context.Background(), c, []Role{role("a", reports{todo{&b}}), role("b", failing{todo{&b}})})
	if err != nil || len(got) != 3 || a || b {
		t.Fatalf("got %+v, a = %v, b = %v, err = %v", got, a, b, err)
	}
	if got[0].Role != Prepare || got[1].Role != "a" || got[2].Role != "b" {
		t.Errorf("roles = %+v", got)
	}
	if items := got[1].Steps[0].Items; len(items) != 1 || items[0].Name != "git" {
		t.Errorf("a's items = %+v", items)
	}
	if items := got[2].Steps[0].Items; len(items) != 1 || items[0].Kind != step.CheckItem {
		t.Errorf("b's failure = %+v, want one check item", items)
	}
}

func TestStateKeepsTheStepsThatOnlyNames(t *testing.T) {
	var done bool
	r := Role{Name: "a", Steps: func(effect.Machine, effect.Units, effect.Out) []step.Step {
		return []step.Step{named{todo{&done}, "x"}, named{todo{&done}, "y"}}
	}}
	for _, tc := range []struct{ only, want []string }{
		{nil, []string{"x", "y"}},
		{[]string{"a"}, []string{"x", "y"}},
		{[]string{"a/y"}, []string{"y"}},
		{[]string{"a/y", "a"}, []string{"x", "y"}},
	} {
		c := Config{Options: Options{Only: tc.only}, M: effecttest.NewMachine(), Out: &effecttest.Out{},
			Prepare: []step.Step{named{todo{&done}, "prepare"}}}
		got, err := State(context.Background(), c, []Role{r})
		if err != nil || len(got) != 2 || len(got[0].Steps) != 1 {
			t.Fatalf("only %q: got %+v, err = %v", tc.only, got, err)
		}
		var steps []string
		for _, report := range got[1].Steps {
			steps = append(steps, report.Step)
		}
		if !slices.Equal(steps, tc.want) {
			t.Errorf("only %q: steps = %q, want %q", tc.only, steps, tc.want)
		}
	}
}

func TestStateStopsAtTheEndOfTheContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var done bool
	r := Role{Name: "a", Steps: func(effect.Machine, effect.Units, effect.Out) []step.Step {
		return []step.Step{todo{&done}}
	}}
	if _, err := State(ctx, Config{M: effecttest.NewMachine(), Out: &effecttest.Out{}}, []Role{r}); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want canceled", err)
	}
}
