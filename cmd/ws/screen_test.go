package main

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestScreenWanted(t *testing.T) {
	for _, tc := range []struct {
		name           string
		plain, in, err bool
		term           string
		want           bool
	}{
		{"terminal", false, true, true, "xterm-256color", true},
		{"plain flag", true, true, true, "xterm-256color", false},
		{"stdin piped", false, false, true, "xterm", false},
		{"stderr piped", false, true, false, "xterm", false},
		{"no TERM", false, true, true, "", false},
		{"dumb TERM", false, true, true, "dumb", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := screenWanted(tc.plain, tc.in, tc.err, tc.term); got != tc.want {
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
