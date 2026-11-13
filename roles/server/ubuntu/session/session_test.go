package session_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/session/sessiontest"
)

func TestSessionStatus(t *testing.T) {
	m := effecttest.NewMachine()
	s := sessiontest.On(m, session.Status{Claude: "ok"})
	st, ok := s.Status(context.Background())
	if !ok {
		t.Fatal("status read nothing")
	}
	if st.Run.Exit != "" || st.Claude != "ok" {
		t.Errorf("status = %+v", st)
	}
	s.Status(context.Background())
	if n := strings.Count(strings.Join(m.Calls, "\n"), "op status"); n != 1 {
		t.Errorf("status ran %d times; want 1", n)
	}
}

func TestSessionStatusFails(t *testing.T) {
	m := effecttest.NewMachine()
	s := sessiontest.On(m, session.Status{})
	m.Results[sessiontest.Op("status")] = effecttest.Result{Err: errors.New("denied")}
	if st, ok := s.Status(context.Background()); ok || st != (session.Status{}) {
		t.Error("status read from a failed ssh")
	}
}
