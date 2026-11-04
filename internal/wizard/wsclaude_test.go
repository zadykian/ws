package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/step"
)

func TestWsClaude(t *testing.T) {
	for name, tc := range map[string]struct{ status, want string }{
		"logged in": {"claude ok\n", ""},
		"no login":  {"claude login\n", "would run claude auth login"},
		"none":      {"claude none\n", "ws lacks root's claude"},
	} {
		r, _ := newRun("lan", tc.status, false, false)
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
	r, m := newRun("lan", "claude login\n", false, false)
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
