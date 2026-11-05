package apply

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/secrets"
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

func TestRun(t *testing.T) {
	for _, check := range []bool{false, true} {
		var a, b bool
		c := Config{Options: Options{Check: check}, M: effecttest.NewMachine(), Out: &effecttest.Out{}}
		got, err := Run(context.Background(), c, []roles.Role{role("a", todo{&a}), role("b", todo{&b})})
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
			list := []roles.Role{a}
			if slices.Contains(tt.only, "b") {
				list = append(list, b)
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
func role(name string, list ...step.Step) roles.Role {
	return roles.Role{Name: name, Steps: func(values.Values, secrets.Secrets, effect.Machine, effect.Out) []step.Step {
		return list
	}}
}

func TestRunNamesEachRoleAndEndsThoseAFailureStopsShortOf(t *testing.T) {
	var done bool
	ok, bad := todo{&done}, failing{todo{&done}}
	tests := []struct {
		name    string
		prepare step.Step
		list    []roles.Role
		want    lasts
	}{
		{"a role fails", ok, []roles.Role{role("a", bad), role("b", ok)},
			lasts{"prepare/todo done", "a/todo failed", "b/todo skipped"}},
		{"prepare fails", bad, []roles.Role{role("a", ok)}, lasts{"prepare/todo failed", "a/todo skipped"}},
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
	changed, err := Run(context.Background(), c, []roles.Role{role("a", todo{&a}), role("b", todo{&b}), role("c", todo{&b})})
	if !errors.Is(err, stopped) || !slices.Equal(changed, []string{"a"}) || b {
		t.Errorf("changed = %q, err = %v, b = %v", changed, err, b)
	}
	if want := (lasts{"a/todo done", "b/todo skipped", "c/todo skipped"}); !slices.Equal(got, want) {
		t.Errorf("last events = %q, want %q", got, want)
	}
}

func TestRunMakesTheSecretsTheWholeTruthWhenGivenAFile(t *testing.T) {
	for _, file := range []string{"", "/s.age"} {
		var whole bool
		r := roles.Role{Name: "a", Steps: func(_ values.Values, s secrets.Secrets, _ effect.Machine, _ effect.Out) []step.Step {
			whole = s.Whole
			return nil
		}}
		c := Config{Options: Options{Secrets: file}, M: effecttest.NewMachine(), Out: &effecttest.Out{}}
		if _, err := Run(context.Background(), c, []roles.Role{r}); err != nil || whole != (file != "") {
			t.Errorf("--secrets %q: whole = %v, err = %v", file, whole, err)
		}
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
