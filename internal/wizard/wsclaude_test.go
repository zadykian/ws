package wizard

import (
	"context"
	"strings"
	"testing"
)

func TestWsClaude(t *testing.T) {
	for name, tc := range map[string]struct{ claude, want string }{
		"logged in": {"ok", ""},
		"no login":  {"login", "would run claude auth login"},
		"none":      {"none", "ws lacks WS_USER's claude"},
	} {
		r, _ := newRun("lan", Status{Claude: tc.claude}, false, false)
		p, err := (&WsClaude{Ws: r.Ws, Tty: true}).Plan(context.Background())
		got := errText(err)
		for _, c := range p.Changes {
			got += c.String()
		}
		if (tc.want == "") != (got == "") || !strings.Contains(got, tc.want) {
			t.Errorf("%s: Plan = %q; want %q", name, got, tc.want)
		}
	}
}
