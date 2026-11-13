package wizard

import (
	"context"
	"github.com/zadykian/ws/engine/core/remote"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/session/sessiontest"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

// macSession returns the Mac's session over ssh, whose remote script answers status on m with the
// value's text.
func macSession(m *effecttest.Machine, ssh remote.SSH, out *effecttest.Out, status session.Status) *session.Session {
	ws := session.New(sessiontest.Reach{Reach: MacReach{SSH: ssh, Conf: "/h/mac.conf", Out: out}, M: m},
		session.Scripts{Remote: []byte("x"), Files: []byte("y")})
	m.Results[sessiontest.Op("status")] = effecttest.Result{Out: sessiontest.Text(status)}
	return ws
}

func TestWsClaudeLoginTakesTheTerminal(t *testing.T) {
	m := effecttest.NewMachine()
	ws := macSession(m, remote.SSH{M: m, F: m, Lock: "/l"}, &effecttest.Out{}, session.Status{Claude: "login"})
	c := &WsClaude{Ws: ws, Tty: true}
	_ = c.Apply(context.Background(), step.Plan{})
	got := callsWith(m, `claude auth login`)
	if len(got) != 1 || !strings.HasPrefix(got[0], "foreground ssh ") {
		t.Errorf("login calls = %q, want one by foreground", got)
	}
}

func TestWsClaudeApply(t *testing.T) {
	r, m := newRun("lan", session.Status{Claude: "login"}, false, false)
	if err := (&WsClaude{Ws: r.Ws}).Apply(context.Background(), step.Plan{}); err == nil ||
		!strings.Contains(err.Error(), "no terminal") {
		t.Errorf("Apply without a terminal = %v", err)
	}
	c := &WsClaude{Ws: r.Ws, Tty: true}
	if err := c.Apply(context.Background(), step.Plan{}); err == nil || !strings.Contains(err.Error(), "still has no login") {
		t.Errorf("Apply where ssh fails = %v", err)
	}
	if !strings.Contains(strings.Join(m.Calls, "\n"), `ws exec bash -lc "claude auth login"`) {
		t.Errorf("calls = %q; want the login", m.Calls)
	}
}
