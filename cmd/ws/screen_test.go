package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/tui"
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

func TestDrawsScreenReadsPlain(t *testing.T) {
	p := process{tty: true, errTTY: true, getenv: func(string) string { return "xterm" }}
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
	err := drawRun(c, fakeProcess(), "title", nil, func(ctx context.Context, b *tui.Bridge) error {
		got = screenMachine(ctx, b, effect.OSMachine{Sudo: []string{}, Confine: "/home/z"})
		if fake := effecttest.NewMachine(); screenMachine(ctx, b, fake) != effect.Machine(fake) {
			return errors.New("a fake changed")
		}
		return nil
	})
	m, ok := got.(effect.OSMachine)
	if err != nil || !ok {
		t.Fatalf("machine = %T, err = %v", got, err)
	}
	if m.Sudo == nil || len(m.Sudo) != 0 || m.Ticket != nil || m.Confine != "/home/z" {
		t.Errorf("sudo = %q, a ticket = %v, confine = %q", m.Sudo, m.Ticket != nil, m.Confine)
	}
	if m.Stdin != effect.NoStdin || m.Stderr == nil {
		t.Errorf("stdin = %v, stderr = %v, want a command off the terminal", m.Stdin, m.Stderr)
	}
}

// The second read passes no bridge, which would fail a second making of the machine.
func TestReadMachineTakesARunsEffectsOnce(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	machine := readMachine(effect.OSMachine{})
	for _, b := range []*tui.Bridge{{}, nil} {
		m, ok := machine(ctx, b).(effect.OSMachine)
		if !ok || m.Stdin != effect.NoStdin || m.Ticket == nil {
			t.Errorf("machine = %+v", m)
		}
	}
	fake := effecttest.NewMachine()
	if got := readMachine(fake)(ctx, nil); got != effect.Machine(fake) {
		t.Errorf("a fake became %T", got)
	}
}
