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

// passMachine records the playbook's run.
type passMachine struct{ *effecttest.Machine }

func (m passMachine) Pass(_ context.Context, _ bool, name string, args ...string) error {
	m.Calls = append(m.Calls, "run "+strings.Join(append([]string{name}, args...), " "))
	return nil
}

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
	var done bool
	ported := func(values.Values, effect.Machine, effect.Out) []step.Step { return []step.Step{todo{&done}} }
	list := []roles.Role{{Name: "a"}, {Name: "b"}, {Name: "c", Steps: ported}, {Name: "d"}}
	segs, err := Segments(list, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		check bool
		want  []string
	}{
		{"run", false, []string{"a", "b", "c", "d"}},
		{"check", true, []string{"a", "b", "c", "d"}},
	} {
		done = false
		m := passMachine{effecttest.NewMachine()}
		c := Config{Options: Options{Check: tc.check, Args: []string{"--skip-tags", "systemd"}}, M: m, Out: &effecttest.Out{}, Dir: "/d"}
		got, err := Run(context.Background(), c, segs)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("%s: changed = %q, want %q", tc.name, got, tc.want)
		}
		var pass []string
		for _, call := range m.Calls {
			if strings.HasPrefix(call, "run env ") || strings.HasPrefix(call, "root env ") {
				pass = append(pass, call)
			}
		}
		if len(pass) != 2 || !strings.Contains(pass[0], "site.yml --tags a,b ") || !strings.Contains(pass[1], "site.yml --tags d ") ||
			!strings.HasSuffix(pass[0], "--skip-tags systemd") || strings.Contains(pass[0], "--check --diff") != tc.check {
			t.Errorf("%s: playbook runs = %q", tc.name, pass)
		}
	}
}
