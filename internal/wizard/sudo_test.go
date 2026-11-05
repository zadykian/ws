package wizard

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/effect/effecttest"
)

const sudoPassword = "a pass'word $x"

// sudoMachine answers sudo's check of a password as sudo-rs does: Results can't, as stdin isn't in
// their key.
type sudoMachine struct{ *effecttest.Machine }

func (m sudoMachine) RunInput(ctx context.Context, stdin []byte, name string, args ...string) ([]byte, error) {
	if strings.HasSuffix(strings.Join(args, " "), "ws sudo -k -S -p '' true") && string(stdin) != sudoPassword+"\n" {
		_, _ = m.Machine.RunInput(ctx, stdin, name, args...)
		return nil, errors.New("[ssh]: exit status 1: \nsudo: Authentication failed, try again.\n\n" +
			"sudo: Authentication required but not attempted\n")
	}
	return m.Machine.RunInput(ctx, stdin, name, args...)
}

// sudoLine is the ssh command line that runs cmd on ws.
func sudoLine(s *Sudo, cmd ...string) string {
	return "ssh " + strings.Join(s.Host.SSH.argv(context.Background(), append([]string{"-o", "BatchMode=yes", "ws"}, cmd...)), " ")
}

// newSudo returns the stage over newHost's machine, where ssh logs in as zadykian, whose sudo needs
// a password, and answers the hidden questions from the list.
func newSudo(t *testing.T, hidden ...string) (*Sudo, *effecttest.Machine, *effecttest.Prompt) {
	t.Helper()
	h, m := newHost(nil)
	m.Results[wayCall(h.SSH)] = effecttest.Result{Out: []byte("user zadykian\nhostname 10.1.1.2\nport 22\n" +
		"userknownhostsfile ~/.ssh/known_hosts\n")}
	h.SSH.M = sudoMachine{m}
	p := &effecttest.Prompt{T: t, Hidden: hidden}
	s := &Sudo{Host: h, P: p}
	m.Results[sudoLine(s, "sudo", "-n", "true")] = effecttest.Result{
		Err: errors.New("[ssh]: exit status 1: sudo: interactive authentication is required\n")}
	m.Results[sudoLine(s, "sudo", "-k", "-S", "-p", "''", "true")] = effecttest.Result{}
	return s, m, p
}

func TestSudoRootLoginAsksNothing(t *testing.T) {
	s, m, _ := newSudo(t)
	m.Results[wayCall(s.Host.SSH)] = effecttest.Result{Out: []byte("user root\nhostname 10.1.1.2\nport 22\n" +
		"userknownhostsfile ~/.ssh/known_hosts\n")}
	p, err := s.Plan(context.Background())
	if err != nil || len(p.Changes) != 0 || s.way != sudoLogin {
		t.Fatalf("Plan = %v, %v, way %d", p, err, s.way)
	}
}

func TestSudoWithNoPasswordAsksNothing(t *testing.T) {
	s, m, _ := newSudo(t)
	m.Results[sudoLine(s, "sudo", "-n", "true")] = effecttest.Result{}
	p, err := s.Plan(context.Background())
	if err != nil || len(p.Changes) != 0 || s.way != sudoOpen {
		t.Fatalf("Plan = %v, %v, way %d", p, err, s.way)
	}
}

func TestSudoStopsWhereSSHFails(t *testing.T) {
	s, m, prompt := newSudo(t)
	m.Results[sudoLine(s, "sudo", "-n", "true")] = effecttest.Result{Err: sshFailed("timed out\r\n")}
	if _, err := s.Plan(context.Background()); err == nil || !strings.Contains(err.Error(), "timed out") ||
		len(prompt.Asked) != 0 {
		t.Fatalf("Plan = %v, Asked %q", err, prompt.Asked)
	}
}

var _ effect.Hidden = (*effecttest.Prompt)(nil)
