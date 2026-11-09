package wizard

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/step"
)

func TestWsClaude(t *testing.T) {
	for name, tc := range map[string]struct{ claude, want string }{
		"logged in": {"ok", ""},
		"no login":  {"login", "would run claude auth login"},
		"none":      {"none", "ws lacks WS_USER's claude"},
	} {
		r, _ := newRun("lan", wsStatus{Claude: tc.claude}, false, false)
		p, err := (&WsClaude{Run: r, Tty: true}).Plan(context.Background())
		got := errText(err)
		for _, c := range p.Changes {
			got += c.String()
		}
		if (tc.want == "") != (got == "") || !strings.Contains(got, tc.want) {
			t.Errorf("%s: Plan = %q; want %q", name, got, tc.want)
		}
	}
}

func TestWsClaudeApply(t *testing.T) {
	r, m := newRun("lan", wsStatus{Claude: "login"}, false, false)
	if err := (&WsClaude{Run: r}).Apply(context.Background(), step.Plan{}); err == nil ||
		!strings.Contains(err.Error(), "no terminal") {
		t.Errorf("Apply without a terminal = %v", err)
	}
	c := &WsClaude{Run: r, Tty: true}
	if err := c.Apply(context.Background(), step.Plan{}); err == nil || !strings.Contains(err.Error(), "still has no login") {
		t.Errorf("Apply where ssh fails = %v", err)
	}
	if !strings.Contains(strings.Join(m.Calls, "\n"), `ws exec bash -lc "claude auth login"`) {
		t.Errorf("calls = %q; want the login", m.Calls)
	}
}

// On ws, the login runs by runuser as host_vars' WS_USER, and a missing claude names ws server apply.
func TestWsClaudeHere(t *testing.T) {
	r, m := newRun("lan", wsStatus{Claude: "none"}, false, false)
	r.Want = func() (conf.Conf, error) { return conf.Conf{User: "ada"}, nil }
	c := &WsClaude{Run: r, Tty: true, Here: m}
	if _, err := c.Plan(context.Background()); err == nil || !strings.Contains(err.Error(), "run sudo ws server apply") {
		t.Errorf("Plan where claude is missing = %v", err)
	}
	m.Results["runuser -l ada -c claude auth login"] = effecttest.Result{}
	if err := c.Apply(context.Background(), step.Plan{}); err != nil ||
		!slices.Contains(m.Calls, "foreground runuser -l ada -c claude auth login") {
		t.Errorf("Apply = %v; calls %q", err, m.Calls)
	}
}
