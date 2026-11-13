package cli_test

import (
	"slices"
	"testing"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/cli/clitest"
)

func TestCallOfReadsPlainAndTheEnvironment(t *testing.T) {
	e := clitest.WithEnv(clitest.Env(), map[string]string{"TERM": "xterm"})
	e.TTY, e.ErrTTY = true, true
	var seen []bool
	root := cli.Root(e)
	root.AddCommand(cli.Group("probe", "x"))
	probe, _, _ := root.Find([]string{"probe"})
	probe.RunE = func(c *cobra.Command, _ []string) error {
		call := cli.CallOf(c, e)
		if call.Path != "ws probe" || call.Getenv("TERM") != "xterm" || !call.TTY || call.OutTTY {
			t.Errorf("call = %+v", call)
		}
		seen = append(seen, call.Plain)
		return nil
	}
	for _, args := range [][]string{{"probe"}, {"probe", "--plain"}, {"--plain", "probe"}} {
		if code := cli.Execute(t.Context(), root, args, nil, nil, nil); code != 0 {
			t.Fatalf("%v exited %d", args, code)
		}
	}
	if want := []bool{false, true, true}; len(seen) != 3 || seen[0] != want[0] || seen[1] != want[1] || seen[2] != want[2] {
		t.Errorf("Plain = %v, want %v", seen, want)
	}
}

func TestCallOfReadsCheckAndOnly(t *testing.T) {
	e := clitest.Env()
	root := cli.Root(e)
	root.AddCommand(cli.Group("probe", "x"))
	probe, _, _ := root.Find([]string{"probe"})
	probe.Flags().Bool("check", false, "")
	probe.Flags().StringSlice("only", nil, "")
	var got []bool
	probe.RunE = func(c *cobra.Command, _ []string) error {
		call := cli.CallOf(c, e)
		got = append(got, call.Check, call.Selected)
		return nil
	}
	for _, args := range [][]string{{"probe"}, {"probe", "--check", "--only", "a"}} {
		if code := cli.Execute(t.Context(), root, args, nil, nil, nil); code != 0 {
			t.Fatalf("%v exited %d", args, code)
		}
	}
	if want := []bool{false, false, true, true}; !slices.Equal(got, want) {
		t.Errorf("Check and Selected = %v, want %v", got, want)
	}
}
