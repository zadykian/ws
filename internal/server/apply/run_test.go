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
		role := func(name string, done *bool) roles.Role {
			return roles.Role{Name: name, Steps: func(values.Values, secrets.Secrets, effect.Machine, effect.Out) []step.Step {
				return []step.Step{todo{done}}
			}}
		}
		c := Config{Options: Options{Check: check}, M: effecttest.NewMachine(), Out: &effecttest.Out{}}
		got, err := Run(context.Background(), c, []roles.Role{role("a", &a), role("b", &b)})
		if err != nil || !slices.Equal(got, []string{"a", "b"}) || a == check || b == check {
			t.Errorf("check %v: changed = %q, a = %v, b = %v, err = %v", check, got, a, b, err)
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

func TestRunNamesEachRoleAndEndsThoseAFailureStopsShortOf(t *testing.T) {
	role := func(name string, s step.Step) roles.Role {
		return roles.Role{Name: name, Steps: func(values.Values, secrets.Secrets, effect.Machine, effect.Out) []step.Step {
			return []step.Step{s}
		}}
	}
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
