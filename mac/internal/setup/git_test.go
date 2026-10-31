package setup_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/conf"
	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/setup"
	"github.com/zadykian/ws/mac/internal/step"
)

const gitHome = "/Users/me"

func gitStep(m *effecttest.Machine, look func(string) (string, error)) *setup.Git {
	if look == nil {
		look = func(name string) (string, error) { return "/opt/bin/" + name, nil }
	}
	return &setup.Git{
		Conf: conf.Conf{GitName: "A B", GitEmail: "a@b.c"},
		Home: gitHome, M: m, Out: &effecttest.Out{}, LookPath: look,
	}
}

func TestGitWithoutNameLeavesGitAlone(t *testing.T) {
	p, err := (&setup.Git{M: effecttest.NewMachine(), Out: &effecttest.Out{}}).Plan(context.Background())
	if err != nil || len(p.Changes) != 0 {
		t.Errorf("Plan = %v, %v; want nothing", p.Changes, err)
	}
}

func TestGitPlansGh(t *testing.T) {
	missing := func(n string) (string, error) {
		if n == "gh" {
			return "", errors.New("not found")
		}
		return "/opt/bin/brew", nil
	}
	for name, tc := range map[string]struct {
		look func(string) (string, error)
		want int
		err  string
	}{
		"gh there":    {nil, 0, ""},
		"no gh":       {missing, 1, ""},
		"no brew":     {func(string) (string, error) { return "", errors.New("none") }, 0, "install Homebrew first"},
		"relative gh": {func(string) (string, error) { return "bin/gh", nil }, 0, "drop PATH's relative directories"},
	} {
		m := effecttest.NewMachine()
		m.Results["brew --prefix"] = effecttest.Result{Out: []byte("/opt\n")}
		p, err := gitStep(m, tc.look).Plan(context.Background())
		if (err == nil) != (tc.err == "") || (err != nil && !strings.Contains(err.Error(), tc.err)) {
			t.Errorf("%s: err = %v, want %q", name, err, tc.err)
		} else if len(p.Changes) != tc.want {
			t.Errorf("%s: changes = %v, want %d", name, p.Changes, tc.want)
		}
	}
}

func TestGitApplyInstallsGh(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["brew install gh"] = effecttest.Result{}
	install := setup.Run{Name: "brew", Args: []string{"install", "gh"}}
	err := gitStep(m, nil).Apply(context.Background(), step.Plan{Changes: []step.Change{install}})
	if err != nil || len(m.Calls) != 1 {
		t.Errorf("Apply = %v, calls %q; want one call", err, m.Calls)
	}
}
