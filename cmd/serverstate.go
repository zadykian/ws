package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine"
	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
)

// newStateCmd returns ws server ubuntu state.
func newStateCmd(p cli.Env) *cobra.Command {
	var opts role.Options
	var asJSON bool
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
		RunE: func(c *cobra.Command, _ []string) error { return runState(c, p, applyOptions{Options: opts}, asJSON) },
	}
	f := c.Flags()
	f.BoolVar(&asJSON, "json", false, "print the report as JSON")
	f.StringSliceVar(&opts.Only, "only", nil, "report the named roles and ROLE/STEPs alone, as a list or by repeating the flag")
	return c
}

// runState prints what the roles report.
func runState(c *cobra.Command, p cli.Env, opts applyOptions, asJSON bool) error {
	r, err := prepareServer(c, p, opts)
	if err != nil {
		return err
	}
	// State reads the machine and takes no secrets. A step that needs them would write them out to
	// plan, as ikeward's install does, and a report has no use for them.
	list, err := r.roles(secrets.Secrets{Unread: true})
	if err != nil {
		return engine.UsageError{Err: err}
	}
	if engine.DrawsReport(cli.CallOf(c, p)) {
		p.Machine = r.cfg.M // ws_home confines it.
		return tui.RunState(c.Context(), engine.ScreenConfig(cli.CallOf(c, p)), c.CommandPath(), func(ctx context.Context, b *tui.Bridge) ([]tui.StateSection, error) {
			// The read takes the screen's effects as a run does, as the steps don't own the terminal under it.
			states, err := role.State(ctx, onScreen(r.cfg, b), list)
			return engine.RoleSections(states), err
		})
	}
	states, err := role.State(c.Context(), r.cfg, list)
	r.term.Flush()
	if err != nil {
		return err
	}
	if asJSON {
		return engine.WriteJSON(c.OutOrStdout(), role.StateDoc{Version: p.Version, Roles: states})
	}
	return engine.WriteRoles(c.OutOrStdout(), states)
}
