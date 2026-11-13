package engine

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/local"
	"github.com/zadykian/ws/engine/tui"
)

func TestScreenWanted(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		plain, json, in, err bool
		term                 string
		want                 bool
	}{
		{"terminal", false, false, true, true, "xterm-256color", true},
		{"plain flag", true, false, true, true, "xterm-256color", false},
		{"stdin piped", false, false, false, true, "xterm", false},
		{"stderr piped", false, false, true, false, "xterm", false},
		{"no TERM", false, false, true, true, "", false},
		{"json", false, true, true, true, "xterm", false},
		{"dumb TERM", false, false, true, true, "dumb", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := screenWanted(tc.plain, tc.json, tc.in, tc.err, tc.term); got != tc.want {
				t.Errorf("screenWanted = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDrawsReportNeedsStdoutAtATerminal(t *testing.T) {
	for _, atTerminal := range []bool{true, false} {
		c := Call{TTY: true, ErrTTY: true, OutTTY: atTerminal, Getenv: func(string) string { return "xterm" }}
		if got := DrawsReport(c); got != atTerminal {
			t.Errorf("stdout at a terminal %v: DrawsReport = %v", atTerminal, got)
		}
	}
}

func TestScreenMachineKeepsRootsSudoAndTheConfine(t *testing.T) {
	var got effect.Machine
	c := Call{
		In: strings.NewReader(""), Err: io.Discard, Out: io.Discard, Getenv: func(string) string { return "" },
		Machine: local.OSMachine{Sudo: []string{}, Confine: "/home/z"},
	}
	err := drawRun(t.Context(), c, "title", nil, func(_ context.Context, b *tui.Bridge) error {
		got = b.Effects().M
		return nil
	})
	m, ok := got.(local.OSMachine)
	if err != nil || !ok {
		t.Fatalf("machine = %T, err = %v", got, err)
	}
	if m.Sudo == nil || len(m.Sudo) != 0 || m.Ticket != nil || m.Confine != "/home/z" {
		t.Errorf("sudo = %q, a ticket = %v, confine = %q", m.Sudo, m.Ticket != nil, m.Confine)
	}
	if m.Stdin != local.NoStdin || m.Stderr == nil {
		t.Errorf("stdin = %v, stderr = %v, want a command off the terminal", m.Stdin, m.Stderr)
	}
}
