package cli

import (
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
)

// procedure returns a command that settles the resources that rs reads, with --check: a role's
// own procedure, which a run state runs, and runs with --check as its check. Its lines go to
// stderr, which the run state passes on as it runs and drops as it checks.
func procedure(use, short, long string, rs func(context.Context, effect.Out) ([]resource.Resource, error)) *cobra.Command {
	c := &cobra.Command{Use: use, Short: short, Long: long + "\n\n--check fails where a change is due, and changes nothing.",
		Args: cobra.NoArgs}
	check := c.Flags().Bool("check", false, "fail where a change is due, and change nothing")
	c.RunE = func(c *cobra.Command, _ []string) error {
		term := effect.NewTerminal(strings.NewReader(""), c.ErrOrStderr())
		defer term.Flush()
		out := linesOf(term, c.CommandPath())
		list, err := rs(c.Context(), out)
		if err != nil {
			return err
		}
		return settle(c.Context(), out, list, *check)
	}
	return c
}
