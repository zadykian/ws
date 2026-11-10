package wizard

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/step"
)

func TestSetup(t *testing.T) {
	for name, tc := range map[string]struct {
		brew, sudo, changed bool
		want                string
	}{
		"no brew": {false, true, true, "after Homebrew"},
		"no sudo": {true, false, true, "sudo -v failed"},
		"changes": {true, true, true, "would apply the steps above"},
		"nothing": {true, true, false, ""},
	} {
		m := effecttest.NewMachine()
		m.Results["sudo -v"] = effecttest.Result{}
		if !tc.sudo {
			m.Results["sudo -v"] = effecttest.Result{Err: errors.New("x")}
		}
		look := func(string) (string, error) { return "/bin/brew", nil }
		if !tc.brew {
			look = func(string) (string, error) { return "", errors.New("none") }
		}
		var checks []bool
		s := &Setup{M: m, LookPath: look, Run: func(_ context.Context, check bool) (bool, error) {
			checks = append(checks, check)
			return tc.changed, nil
		}}
		p, err := s.Plan(context.Background())
		got := errText(err)
		for _, c := range p.Changes {
			got += c.String()
		}
		if (tc.want == "") != (got == "") || !strings.Contains(got, tc.want) {
			t.Errorf("%s: Plan = %q, want %q", name, got, tc.want)
		}
	}
}

func TestStagesGiveTheWordsOfNoTicket(t *testing.T) {
	m := effect.OSMachine{Ticket: func() error { return effect.ErrNoTicket }}
	_, planned := (&Setup{M: m, LookPath: func(string) (string, error) { return "", nil }}).Plan(context.Background())
	tun, _ := newTunnel(keyA, wsStatus{}, false)
	tun.M = m
	if err := tun.Apply(context.Background(), step.Plan{}); !errors.Is(err, effect.ErrNoTicket) || !errors.Is(planned, err) {
		t.Errorf("Apply = %v, Plan = %v", err, planned)
	}
}

func TestSetupApply(t *testing.T) {
	var checks []bool
	s := &Setup{Run: func(_ context.Context, check bool) (bool, error) {
		checks = append(checks, check)
		return true, errors.New("stopped")
	}}
	if err := s.Apply(context.Background(), step.Plan{}); err == nil || len(checks) != 1 || checks[0] {
		t.Errorf("Apply = %v, checks %v; want the error of a run with check unset", err, checks)
	}
}

func TestSetupIsDoneOnceApplied(t *testing.T) {
	// ws mac apply's own plan still finds a change after it ran: a step that only warned, as cld's install.
	s := &Setup{M: effecttest.NewMachine(), LookPath: func(string) (string, error) { return "/brew", nil },
		Run: func(context.Context, bool) (bool, error) { return true, nil }}
	s.sudoed = true
	if p, err := s.Plan(context.Background()); err != nil || len(p.Changes) != 1 {
		t.Fatalf("Plan before Apply = %v, %v; want the change", p, err)
	}
	if err := s.Apply(context.Background(), step.Plan{}); err != nil {
		t.Fatal(err)
	}
	if p, err := s.Plan(context.Background()); err != nil || len(p.Changes) != 0 {
		t.Errorf("Plan after Apply = %v, %v; want none", p, err)
	}
}
