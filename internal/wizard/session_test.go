package wizard

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/conf"
)

// wsSession returns ws's own session, whose host_vars names ada, and whose remote script answers
// status with the value's text.
func wsSession(status wsStatus) (*Session, *effecttest.Machine) {
	m := effecttest.NewMachine()
	want := func() (conf.Conf, error) { return conf.Conf{Target: "lan", User: "ada"}, nil }
	reach := WsReach{M: m, Want: want, Here: m, Out: &effecttest.Out{}}
	ws := NewSession(fakeReach{Reach: reach, M: m}, Scripts{Remote: []byte("x"), Files: []byte("y")})
	m.Results[opLine("status")] = effecttest.Result{Out: status.text()}
	return ws, m
}

func TestSessionStatus(t *testing.T) {
	m := effecttest.NewMachine()
	s := macSession(m, SSH{M: m, Lock: "/l"}, &effecttest.Out{}, wsStatus{Claude: "ok"})
	st, ok := s.status(context.Background())
	if !ok {
		t.Fatal("status read nothing")
	}
	if st.Run.Exit != "" || st.Claude != "ok" {
		t.Errorf("status = %+v", st)
	}
	s.status(context.Background())
	if n := strings.Count(strings.Join(m.Calls, "\n"), "op status"); n != 1 {
		t.Errorf("status ran %d times; want 1", n)
	}
}

func TestSessionStatusFails(t *testing.T) {
	m := effecttest.NewMachine()
	s := macSession(m, SSH{M: m, Lock: "/l"}, &effecttest.Out{}, wsStatus{})
	m.Results[opLine("status")] = effecttest.Result{Err: errors.New("denied")}
	if st, ok := s.status(context.Background()); ok || st != (wsStatus{}) {
		t.Error("status read from a failed ssh")
	}
}
