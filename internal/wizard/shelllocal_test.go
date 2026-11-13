package wizard

import (
	"context"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

// localSession returns ws's own session over m, whose runs' events go to events.
func localSession(m effect.Machine, events step.Observer) *session.Session {
	want := func() (session.Want, error) { return session.Want{}, nil }
	return session.New(session.WsReach{M: m, Read: want}, session.Scripts{Remote: []byte("echo\n"), Events: events})
}

// On ws, the script goes on sh's stdin and each word goes unchanged, with no ssh and no sudo.
func TestSessionOnWsRunsTheScriptLocally(t *testing.T) {
	m := effecttest.NewMachine()
	r := localSession(m, nil)
	m.Results["sh -s -- status"] = effecttest.Result{Out: []byte("ws abc\n")}
	if s, ok := r.Status(context.Background()); !ok || s.Version != "abc" {
		t.Fatalf("status read %q; calls %q", s.Version, m.Calls)
	}
	m.Results[session.WsPath+" server ubuntu secrets set env A'B"] = effecttest.Result{Out: []byte("set\n")}
	if out, err := localSession(m, nil).Secrets(context.Background(), []byte("dummy-value\n"), "set", "env", "A'B"); err != nil || string(out) != "set\n" {
		t.Fatalf("Secrets = %q, %v", out, err)
	}
	if want := []string{"echo\n", "dummy-value\n"}; !slices.Equal(m.Inputs, want) {
		t.Errorf("stdin = %q, want %q", m.Inputs, want)
	}
}

func TestFollowOnWsHandsEventsOn(t *testing.T) {
	m := effecttest.NewMachine()
	h := &heard{}
	r := localSession(m, h)
	m.Results["sh -s -- run --only=signoz/signoz up"] = effecttest.Result{Out: []byte(runLog)}
	if err := r.Run(context.Background(), []string{"--only=signoz/signoz up"}); err != nil {
		t.Fatal(err)
	}
	if got := h.steps(); !slices.Equal(got, []string{"ssh/keys"}) {
		t.Errorf("events = %q", got)
	}
	if want := []string{"ssh: changed", "last line"}; !slices.Equal(m.Passed, want) {
		t.Errorf("Passed = %q, want %q", m.Passed, want)
	}
}

// A Machine without effect.Lines still runs the run, its lines all passed on.
func TestFollowOnWsWithoutLines(t *testing.T) {
	m := effecttest.NewMachine()
	h := &heard{}
	r := localSession(struct{ effect.Machine }{m}, h)
	m.Results["sh -s -- run"] = effecttest.Result{Out: []byte(runLog)}
	if err := r.Run(context.Background(), nil); err != nil || len(*h) != 0 {
		t.Errorf("run = %v, events %v", err, *h)
	}
	if !slices.Equal(m.Calls, []string{"run sh -s -- run"}) {
		t.Errorf("calls = %q", m.Calls)
	}
}

// WS_USER reaches the script as SUDO_USER, as sudo names the Mac's ssh login; an empty one leaves it.
func TestLocalShellNamesWsUser(t *testing.T) {
	m := effecttest.NewMachine()
	user := "ada"
	r := session.New(session.WsReach{M: m, Read: func() (session.Want, error) { return session.Want{User: user}, nil }}, session.Scripts{Remote: []byte("echo\n")})
	m.Results["env SUDO_USER=ada sh -s -- status"] = effecttest.Result{Out: []byte("claude ok\n")}
	if s, ok := r.Status(context.Background()); !ok || s.Claude != "ok" {
		t.Errorf("status read %q; calls %q", s.Claude, m.Calls)
	}
	user = ""
	m.Results["sh -s -- install v1 url"] = effecttest.Result{}
	if err := r.Install(context.Background(), "v1", "url"); err != nil || m.Calls[len(m.Calls)-1] != "run sh -s -- install v1 url" {
		t.Errorf("Pass = %v; calls %q", err, m.Calls)
	}
}
