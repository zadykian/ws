package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/local"
	"github.com/zadykian/ws/engine/tui"
)

// screenCommand returns a command that keys type on, and what it draws.
func screenCommand(t *testing.T, keys string) (*cobra.Command, *bytes.Buffer) {
	var drawn bytes.Buffer
	c := &cobra.Command{Use: "apply"}
	c.SetContext(t.Context())
	c.SetIn(strings.NewReader(keys))
	c.SetErr(&drawn)
	return c, &drawn
}

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
		p := cli.Env{TTY: true, ErrTTY: true, OutTTY: atTerminal, Getenv: func(string) string { return "xterm" }}
		if got := drawsReport(&cobra.Command{}, p); got != atTerminal {
			t.Errorf("stdout at a terminal %v: drawsReport = %v", atTerminal, got)
		}
	}
}

func TestDrawsScreenReadsPlain(t *testing.T) {
	p := cli.Env{TTY: true, ErrTTY: true, Getenv: func(string) string { return "xterm" }}
	var seen []bool
	root := newRoot(p)
	root.AddCommand(group("probe", "x"))
	probe, _, _ := root.Find([]string{"probe"})
	probe.RunE = func(c *cobra.Command, _ []string) error { seen = append(seen, drawsScreen(c, p)); return nil }
	for _, args := range [][]string{{"probe"}, {"probe", "--plain"}, {"--plain", "probe"}} {
		if code := execute(t.Context(), root, args, nil, nil, nil); code != 0 {
			t.Fatalf("%v exited %d", args, code)
		}
	}
	if want := []bool{true, false, false}; len(seen) != 3 || seen[0] != want[0] || seen[1] != want[1] || seen[2] != want[2] {
		t.Errorf("drawsScreen = %v, want %v", seen, want)
	}
}

func TestScreenMachineKeepsRootsSudoAndTheConfine(t *testing.T) {
	var got effect.Machine
	c, _ := screenCommand(t, "")
	p := fakeProcess()
	p.Machine = local.OSMachine{Sudo: []string{}, Confine: "/home/z"}
	err := drawRun(c, p, "title", nil, func(_ context.Context, b *tui.Bridge) error {
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
