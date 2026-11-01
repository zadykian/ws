package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

const latestCmd = "run curl -fsS -o /dev/null -w %{redirect_url} http://127.0.0.1:22080/latest"

func newMacUpdate(version, redirect string) (*MacUpdate, *effecttest.Machine, *effecttest.Out) {
	m := effecttest.NewMachine()
	out := &effecttest.Out{}
	m.Results[strings.TrimPrefix(latestCmd, "run ")] = effecttest.Result{Out: []byte(redirect)}
	return &MacUpdate{M: m, Out: out, Version: version, URL: "http://127.0.0.1:22080", Home: "/h",
		Args: []string{"mac", "wizard", "--update"}}, m, out
}

func TestMacUpdatePlan(t *testing.T) {
	for name, tc := range map[string]struct {
		version, redirect string
		changes           int
		warns, fails      bool
	}{
		"latest":      {"v1.2.3", "http://127.0.0.1:22080/tag/v1.2.3", 0, false, false},
		"newer":       {"v1.2.3", "http://127.0.0.1:22080/tag/v1.3.0", 1, false, false},
		"pseudo":      {"v1.2.4-0.20260101000000-abcdef123456", "http://127.0.0.1:22080/tag/v1.3.0", 0, true, false},
		"dirty":       {"v1.2.3+dirty", "http://127.0.0.1:22080/tag/v1.3.0", 0, true, false},
		"devel":       {"devel", "", 0, true, false},
		"no redirect": {"v1.2.3", "", 0, false, true},
	} {
		u, _, out := newMacUpdate(tc.version, tc.redirect)
		p, err := u.Plan(context.Background())
		if len(p.Changes) != tc.changes || (err != nil) != tc.fails || (len(out.Lines) > 0) != tc.warns {
			t.Errorf("%s: Plan = %v, %v, said %q; want %d changes, fails %v, warns %v",
				name, p.Changes, err, out.Lines, tc.changes, tc.fails, tc.warns)
		}
	}
}

func TestMacUpdatePlanRefusesAddress(t *testing.T) {
	u, m, _ := newMacUpdate("v1.2.3", "")
	u.URL = "http://example.com"
	if _, err := u.Plan(context.Background()); err == nil || len(m.Calls) != 0 {
		t.Errorf("Plan with another address = %v, calls %v; want an error and no call", err, m.Calls)
	}
}
