package apply

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/server/roles"
	"github.com/zadykian/ws/internal/server/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

// recapMachine writes the playbook's log as ansible-playbook does, with the changes it names.
type recapMachine struct {
	*effecttest.Machine
	log     string
	changed string
}

func (m *recapMachine) Pass(_ context.Context, _ bool, name string, args ...string) error {
	if m.changed != "" {
		m.Files[m.log] = []byte("2026-11-01 INFO| localhost : ok=3 changed=" + m.changed + " unreachable=0 failed=0\n")
	}
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
	ported := func(values.Values, secrets.Secrets, effect.Machine, effect.Out) []step.Step {
		return []step.Step{todo{&done}}
	}
	list := []roles.Role{{Name: "a"}, {Name: "b"}, {Name: "c", Steps: ported}, {Name: "d"}}
	segs, err := Segments(list, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		check   bool
		changed string
		want    []string
	}{
		{"steps changed", false, "0", []string{"c"}},
		{"playbook changed", false, "2", []string{"a", "b", "c", "d"}},
		{"check", true, "0", []string{"c"}},
	} {
		done = false
		m := &recapMachine{Machine: effecttest.NewMachine(), log: "/log", changed: tc.changed}
		c := Config{Options: Options{Check: tc.check, Args: []string{"--skip-tags", "systemd"}}, M: m, Out: &effecttest.Out{}, Dir: "/d", Log: "/log"}
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

func TestRunWithoutRecap(t *testing.T) {
	segs, _ := Segments([]roles.Role{{Name: "a"}}, nil)
	m := &recapMachine{Machine: effecttest.NewMachine(), log: "/log"}
	c := Config{M: m, Out: &effecttest.Out{}, Dir: "/d", Log: "/log"}
	if _, err := Run(context.Background(), c, segs); err == nil || !strings.Contains(err.Error(), "no recap") {
		t.Errorf("err = %v, want no recap", err)
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
