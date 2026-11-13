package session_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/session/sessiontest"
)

// Each operation leaves the status stale where it changes ws, and else not. Each case reads the
// status, runs the operation, reads the status again, and counts the status ops sent.
func TestEachOperationsStaleRule(t *testing.T) {
	ctx := context.Background()
	for name, tc := range map[string]struct {
		run     func(*session.Session) error
		changes bool
	}{
		"ports":     {func(s *session.Session) error { _, err := s.Ports(ctx); return err }, false},
		"waitPorts": {func(s *session.Session) error { _, err := s.WaitPorts(ctx, "eth1"); return err }, false},
		"nameCable": {func(s *session.Session) error { return s.NameCable(ctx, "eth1") }, true},
		"nameUser":  {func(s *session.Session) error { return s.NameUser(ctx, "ada") }, true},
		"namePeer":  {func(s *session.Session) error { return s.NamePeer(ctx, "key") }, true},
		"install":   {func(s *session.Session) error { return s.Install(ctx, "v1", "url") }, true},
		"run":       {func(s *session.Session) error { return s.Run(ctx, nil) }, true},
		"outbound":  {func(s *session.Session) error { return s.Outbound(ctx) }, true},
		"values":    {func(s *session.Session) error { _, err := s.Values(ctx); return err }, false},
		"setValues": {func(s *session.Session) error { return s.SetValues(ctx, []string{"a=b"}) }, true},
	} {
		m := effecttest.NewMachine()
		s := sessiontest.On(m, session.Status{})
		for _, line := range []string{"link", "link eth1", "wait eth1", "user ada", "peer key", "install v1 url",
			"run", "run --only network,wireguard", "run --only wireguard", "outbound", "values", "values YT1i"} {
			m.Results["op "+line] = effecttest.Result{}
		}
		s.Status(ctx)
		if err := tc.run(s); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		s.Status(ctx)
		want := 1
		if tc.changes {
			want = 2
		}
		if got := strings.Count(strings.Join(m.Calls, "\n"), "op status"); got != want {
			t.Errorf("%s: the status was read %d times; want %d", name, got, want)
		}
	}
}
