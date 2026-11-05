package wizard

import (
	"context"
	"testing"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/effect/effecttest"
)

// newSudo returns the stage over newHost's machine, where ssh logs in as zadykian.
func newSudo(t *testing.T) (*Sudo, *effecttest.Machine) {
	t.Helper()
	h, m := newHost(nil)
	m.Results[wayCall(h.SSH)] = effecttest.Result{Out: []byte("user zadykian\nhostname 10.1.1.2\nport 22\n" +
		"userknownhostsfile ~/.ssh/known_hosts\n")}
	return &Sudo{Host: h, P: &effecttest.Prompt{T: t}}, m
}

func TestSudoRootLoginAsksNothing(t *testing.T) {
	s, m := newSudo(t)
	m.Results[wayCall(s.Host.SSH)] = effecttest.Result{Out: []byte("user root\nhostname 10.1.1.2\nport 22\n" +
		"userknownhostsfile ~/.ssh/known_hosts\n")}
	p, err := s.Plan(context.Background())
	if err != nil || len(p.Changes) != 0 || s.way != sudoLogin {
		t.Fatalf("Plan = %v, %v, way %d", p, err, s.way)
	}
}

func TestSudoStopsOnAnotherLogin(t *testing.T) {
	s, _ := newSudo(t)
	if _, err := s.Plan(context.Background()); err == nil || s.way != sudoUnread {
		t.Fatalf("Plan = %v, way %d", err, s.way)
	}
}

var _ effect.Hidden = (*effecttest.Prompt)(nil)
