package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

// ErrDue is the failure of a command's --check that finds a change to make, which a run state
// takes for a run to do.
var ErrDue = errors.New("to do")

// Procedure returns a command that settles the resources that rs reads, with --check: a role's
// own procedure, which a run state runs, and runs with --check as its check. Each side's roles
// take this one shape. Its lines go to stderr, which the run state passes on as it runs and drops
// as it checks.
func Procedure(use, short, long string, rs func(context.Context, effect.Out) ([]resource.Resource, error)) *cobra.Command {
	c := &cobra.Command{Use: use, Short: short, Long: long + "\n\n--check fails where a change is due, and changes nothing.",
		Args: cobra.NoArgs}
	check := c.Flags().Bool("check", false, "fail where a change is due, and change nothing")
	c.RunE = func(c *cobra.Command, _ []string) error {
		term := effect.NewTerminal(c.ErrOrStderr())
		defer term.Flush()
		out := engine.Lines{Out: term, Prog: c.CommandPath(), Seen: map[string]bool{}}
		list, err := rs(c.Context(), out)
		if err != nil {
			return err
		}
		return Settle(c.Context(), out, list, *check)
	}
	return c
}

// Settle plans each resource of rs, in order. With check, it fails with ErrDue and every change's
// headline where one is due. Otherwise it applies each resource's changes before it plans the next,
// and says each on out.
func Settle(ctx context.Context, out effect.Out, rs []resource.Resource, check bool) error {
	var due []string
	for _, r := range rs {
		changes, err := r.Plan(ctx)
		if err != nil {
			return err
		}
		for _, c := range changes {
			if check {
				due = append(due, step.Headline(c))
				continue
			}
			if err := c.Apply(ctx); err != nil {
				return err
			}
			out.Say("%s", step.Headline(c))
		}
	}
	if len(due) > 0 {
		return fmt.Errorf("%w: %s", ErrDue, strings.Join(due, "; "))
	}
	return nil
}
