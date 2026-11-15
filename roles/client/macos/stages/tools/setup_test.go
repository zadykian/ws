package tools

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/values"
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

func TestSetupWaitsOnTheMacKey(t *testing.T) {
	// Under --check the Mac key stage writes no line, so ws client macos apply's plan would stop for
	// the key.
	for conf, want := range map[string]string{
		values.Text("WS_USER", "ada"): "ws client macos apply: after Mac key",
		values.Text("WS_IDENTITY_PUB_KEY", "~/k.pub", "WS_GIT_SIGNING_PUB_KEY", ""):        "ws client macos apply: after Mac key",
		values.Text("WS_IDENTITY_PUB_KEY", "~/k.pub", "WS_GIT_SIGNING_PUB_KEY", "~/k.pub"): "ws client macos apply: would apply the steps above",
	} {
		m := effecttest.NewMachine()
		m.Results["sudo -v"] = effecttest.Result{}
		m.Files["/h/values.yml"] = []byte(conf)
		s := &Setup{Conf: "/h/values.yml", M: m, LookPath: func(string) (string, error) { return "/brew", nil },
			Run: func(context.Context, bool) (bool, error) { return true, nil }}
		p, err := s.Plan(context.Background())
		if err != nil || len(p.Changes) != 1 || p.Changes[0].String() != want {
			t.Errorf("%q: Plan = %v, %v; want %q", conf, p, err, want)
		}
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
	// ws client macos apply's own plan still finds a change after it ran: a step that only warned, as
	// cld's install.
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

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
