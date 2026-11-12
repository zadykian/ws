package wizard

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// secretsLine is the ssh command line that runs cmd as root on ws, through sudo's password.
func secretsLine(sh SSHShell, cmd string) string {
	return "ssh " + strings.Join(sh.SSH.argv(context.Background(),
		[]string{"-o", "BatchMode=yes", "ws", "sudo", "-k", "-S", "-p", "''", wsPath, "server", "secrets"}), " ") + " " + cmd
}

// shellReach is a Reach whose shell is sh.
type shellReach struct {
	Reach
	sh Shell
}

func (r shellReach) shell() Shell { return r.sh }

// secretsSession returns a session that reaches ws through sh.
func secretsSession(sh Shell) *Session {
	return NewSession(shellReach{sh: sh}, Scripts{Remote: []byte("echo\n")})
}

func TestSessionSecretsSendsTheValueOnStdin(t *testing.T) {
	m := effecttest.NewMachine()
	sh := SSHShell{SSH: SSH{M: m, Lock: "/t/l"}, Sudo: &Sudo{way: sudoAsked, password: "pw"}}
	r := secretsSession(sh)
	m.Results[secretsLine(sh, `set env 'A'\''B'`)] = effecttest.Result{Out: []byte("set\n")}
	out, err := r.Secrets(context.Background(), []byte("dummy-value\n"), "set", "env", "A'B")
	if err != nil || string(out) != "set\n" {
		t.Fatalf("Secrets = %q, %v", out, err)
	}
	if got := m.Inputs[0]; got != "pw\ndummy-value\n" {
		t.Errorf("stdin = %q, want the password, then the value", got)
	}
	if strings.Contains(strings.Join(m.Calls, " "), "dummy-value") {
		t.Error("an argument holds the value")
	}
}

func TestSessionSecretsErrors(t *testing.T) {
	for _, tt := range []struct{ err, want string }{
		{"[ssh]: exit status 255: \nssh: connect to host ws: No route to host\n", "ws is out of reach: ssh to ws failed: ssh: connect"},
		{"[ssh]: exit status 1: \nws server secrets keys: run it as root\n", "ws server secrets keys: run it as root"},
	} {
		m := effecttest.NewMachine()
		sh := SSHShell{SSH: SSH{M: m, Lock: "/t/l"}}
		r := secretsSession(sh)
		m.Results["ssh "+strings.Join(sh.SSH.argv(context.Background(),
			[]string{"-o", "BatchMode=yes", "ws", wsPath, "server", "secrets", "keys"}), " ")] = effecttest.Result{Err: errors.New(tt.err)}
		if _, err := r.Secrets(context.Background(), nil, "keys"); err == nil || !strings.HasPrefix(err.Error(), tt.want) {
			t.Errorf("err = %v, want %q", err, tt.want)
		}
	}
}

// The status stays stale after Secrets, which no table row covers.
func TestSessionSecretsLeavesTheStatusStale(t *testing.T) {
	ctx := context.Background()
	m := effecttest.NewMachine()
	ws := NewSession(fakeReach{Reach: MacReach{}, M: m}, Scripts{Remote: []byte("x")})
	m.Results[opLine("status")] = effecttest.Result{Out: wsStatus{}.text()}
	m.Results[wsPath+" server secrets keys"] = effecttest.Result{}
	ws.status(ctx)
	if _, err := ws.Secrets(ctx, nil, "keys"); err != nil {
		t.Fatal(err)
	}
	ws.status(ctx)
	if got := strings.Count(strings.Join(m.Calls, "\n"), opLine("status")); got != 2 {
		t.Errorf("status read %d times, want 2", got)
	}
}
