package cmd

import (
	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
)

// newStateCmd returns ws server ubuntu state.
func newStateCmd(p cli.Env) *cobra.Command {
	var opts applyOptions
	c := &cobra.Command{
		Use:   "state",
		Short: "report what the server holds of what ws manages, and what apply would change",
		Long: "It reads alone and applies nothing. It needs what --check needs: root, on Ubuntu 26.04. " +
			"It reports what ws manages and not the rest of the machine: the packages, services and files of the " +
			"steps, what stands against them, and the changes that apply would make. A step that fails stands " +
			"in the report as a check, and the exit code stays 0. It takes no secrets, so a report holds none and " +
			"says nothing of what they write.\n\n" +
			"At a terminal it draws the state screen on stderr, where the tabs, a filter and r to read again are. " +
			"--plain prints the report as text, and so does a pipe. The text is on stdout and ws's own lines on stderr. --json prints the version of ws, then " +
			"each role with its steps, items and changes.",
		Args: cobra.NoArgs,
		RunE: cli.State(serverTarget(p, &opts), p),
	}
	f := c.Flags()
	f.Bool("json", false, "print the report as JSON")
	f.StringSliceVar(&opts.Only, "only", nil, "report the named roles and ROLE/STEPs alone, as a list or by repeating the flag")
	return c
}
