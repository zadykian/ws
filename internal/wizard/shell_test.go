package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestShellWord(t *testing.T) {
	for word, want := range map[string]string{
		"status":                     "status",
		"--only=network,wireguard":   "--only=network,wireguard",
		"https://example.test/a+b@c": "https://example.test/a+b@c",
		"":                           "''",
		"signoz up":                  "'signoz up'",
		"it's":                       `'it'\''s'`,
		"$HOME":                      "'$HOME'",
		"a;b":                        "'a;b'",
	} {
		if got := shellWord(word); got != want {
			t.Errorf("shellWord(%q) = %q, want %q", word, got, want)
		}
	}
}

// ssh hands ws's shell one line, which reads back the words that the Shell was given.
func TestSSHShellQuotesForWsShell(t *testing.T) {
	m := effecttest.NewMachine()
	s := SSHShell{SSH: SSH{M: m, Lock: "/t/l"}}
	line := "ssh " + strings.Join(s.SSH.argv(context.Background(),
		[]string{"-o", "BatchMode=yes", "ws", "sh", "-s", "--", "run", "'--only=signoz/signoz up'"}), " ")
	m.Results[line] = effecttest.Result{Out: []byte("ok\n")}
	out, err := s.Run(context.Background(), []byte("script"), "sh", "-s", "--", "run", "--only=signoz/signoz up")
	if err != nil || string(out) != "ok\n" {
		t.Fatalf("Run = %q, %v; calls %q", out, err, m.Calls)
	}
	if m.Inputs[0] != "script" {
		t.Errorf("stdin = %q", m.Inputs[0])
	}
}
