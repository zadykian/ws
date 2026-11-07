package wizard

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

// secretsLine is the ssh command line that runs cmd as root on ws, through sudo's password.
func secretsLine(r *Remote, cmd string) string {
	return "ssh " + strings.Join(r.SSH.argv(context.Background(),
		[]string{"-o", "BatchMode=yes", "ws", "sudo", "-k", "-S", "-p", "''", wsPath, "server", "secrets"}), " ") + " " + cmd
}

func TestRemoteSecretsSendsTheValueOnStdin(t *testing.T) {
	m := effecttest.NewMachine()
	r := &Remote{SSH: SSH{M: m, Lock: "/t/l"}, Sudo: &Sudo{way: sudoAsked, password: "pw"}}
	m.Results[secretsLine(r, `'set' 'env' 'A'\''B'`)] = effecttest.Result{Out: []byte("set\n")}
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

func TestRemoteSecretsErrors(t *testing.T) {
	for _, tt := range []struct{ err, want string }{
		{"[ssh]: exit status 255: \nssh: connect to host ws: No route to host\n", "ws is out of reach: ssh to ws failed: ssh: connect"},
		{"[ssh]: exit status 1: \nws server secrets keys: run it as root\n", "ws server secrets keys: run it as root"},
	} {
		m := effecttest.NewMachine()
		r := &Remote{SSH: SSH{M: m, Lock: "/t/l"}}
		m.Results["ssh "+strings.Join(r.SSH.argv(context.Background(),
			[]string{"-o", "BatchMode=yes", "ws", wsPath, "server", "secrets", "'keys'"}), " ")] = effecttest.Result{Err: errors.New(tt.err)}
		if _, err := r.Secrets(context.Background(), nil, "keys"); err == nil || !strings.HasPrefix(err.Error(), tt.want) {
			t.Errorf("err = %v, want %q", err, tt.want)
		}
	}
}
