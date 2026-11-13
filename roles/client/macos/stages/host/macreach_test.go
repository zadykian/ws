package host

import (
	"context"
	"github.com/zadykian/ws/engine/core/remote"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/session/sessiontest"
	"github.com/zadykian/ws/roles/server/ubuntu/stages"
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
	c := &stages.WsClaude{Ws: ws, Tty: true}
	_ = c.Apply(context.Background(), step.Plan{})
	got := callsWith(m, `claude auth login`)
	if len(got) != 1 || !strings.HasPrefix(got[0], "foreground ssh ") {
		t.Errorf("login calls = %q, want one by foreground", got)
	}
}

func TestWsClaudeApply(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/mac.conf"] = []byte("WS_TARGET=lan\nWS_USER=ada\n")
	ws := macSession(m, remote.SSH{M: m, Lock: "/l", Ways: MacWays{M: m, Conf: "/h/mac.conf"}}, &effecttest.Out{},
		session.Status{Claude: "login"})
	if err := (&stages.WsClaude{Ws: ws}).Apply(context.Background(), step.Plan{}); err == nil ||
		!strings.Contains(err.Error(), "no terminal") {
		t.Errorf("Apply without a terminal = %v", err)
	}
	c := &stages.WsClaude{Ws: ws, Tty: true}
	if err := c.Apply(context.Background(), step.Plan{}); err == nil || !strings.Contains(err.Error(), "still has no login") {
		t.Errorf("Apply where ssh fails = %v", err)
	}
	if !strings.Contains(strings.Join(m.Calls, "\n"), `ws exec bash -lc "claude auth login"`) {
		t.Errorf("calls = %q; want the login", m.Calls)
	}
}

func TestMacReachWantsTheSlotsAddressAndThePort(t *testing.T) {
	for name, tc := range map[string]struct{ file, addr, port string }{
		"no slot":      {"WS_TARGET=lan\n", "10.99.0.1/24", ""},
		"slot 1":       {"WS_TARGET=lan\nWS_SLOT=1\nWS_PORT=51821\n", "10.99.1.1/24", "51821"},
		"last slot":    {"WS_TARGET=lan\nWS_SLOT=15\n", "10.99.15.1/24", ""},
		"port alone":   {"WS_TARGET=lan\nWS_PORT=51820\n", "10.99.0.1/24", "51820"},
		"unset values": {"WS_TARGET=lan\nWS_USER=ada\n", "10.99.0.1/24", ""},
	} {
		m := effecttest.NewMachine()
		m.Files["/h/server.conf"] = []byte(tc.file)
		w, err := MacReach{SSH: remote.SSH{M: m}, Conf: "/h/server.conf"}.Want()
		if err != nil || w.Address.String() != tc.addr || w.Port != tc.port {
			t.Errorf("%s: Want = %v, %q, %v; want %s, %q", name, w.Address, w.Port, err, tc.addr, tc.port)
		}
	}
}
