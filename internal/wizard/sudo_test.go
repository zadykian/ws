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
		Err: errors.New("[ssh]: exit status 1: sudo: interactive authentication is required")}
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

func TestSudoAsksForThePassword(t *testing.T) {
	s, m, prompt := newSudo(t, sudoPassword)
	p, err := s.Plan(context.Background())
	if err != nil || len(p.Changes) != 1 ||
		p.Changes[0].String() != "Sudo: would ask for zadykian's sudo password on ws, at the terminal" {
		t.Fatalf("Plan = %v, %v", p, err)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if len(prompt.Asked) != 1 || prompt.Asked[0] != "zadykian's sudo password on ws" {
		t.Errorf("Asked = %q", prompt.Asked)
	}
	if again, err := s.Plan(context.Background()); err != nil || len(again.Changes) != 0 {
		t.Errorf("Plan after Apply = %v, %v", again, err)
	}
	if s.way != sudoAsked || s.password != sudoPassword {
		t.Errorf("way %d, password %q", s.way, s.password)
	}
	said := strings.Join(s.Host.Out.(*effecttest.Out).Lines, "\n")
	if strings.Contains(said, sudoPassword) || strings.Contains(strings.Join(m.Calls, "\n"), sudoPassword) {
		t.Errorf("the password shows in %q or %q", said, m.Calls)
	}
}

func TestSudoStopsOnAWrongPassword(t *testing.T) {
	s, _, _ := newSudo(t, "one")
	p, _ := s.Plan(context.Background())
	err := s.Apply(context.Background(), p)
	if err == nil || s.way != sudoUnread || s.password != "" {
		t.Fatalf("Apply = %v, way %d, password %q: the stage kept a refused password", err, s.way, s.password)
	}
}

func TestSudoStopsOnAUserOutsideSudoers(t *testing.T) {
	s, m, _ := newSudo(t, sudoPassword)
	m.Results[sudoLine(s, "sudo", "-k", "-S", "-p", "''", "true")] = effecttest.Result{
		Err: errors.New("[ssh]: exit status 1: sudo: I'm sorry zadykian. I'm afraid I can't do that")}
	p, _ := s.Plan(context.Background())
	err := s.Apply(context.Background(), p)
	if err == nil || !strings.Contains(err.Error(), "afraid") || strings.Contains(err.Error(), sudoPassword) {
		t.Fatalf("Apply = %v", err)
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
