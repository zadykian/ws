package cmd

import (
	"errors"

	"github.com/spf13/cobra"
)

// newServerWizardCmd returns ws server wizard.
func newServerWizardCmd(p process) *cobra.Command {
	var o wizardOptions
	c := &cobra.Command{
		Use:   "wizard",
		Short: "set ws up from its own shell, in stages",
		Long: "The stages of ws mac wizard that act on ws, run as root on ws: ws's values, which it asks where " +
			"host_vars lacks them, then ws server apply, in a unit of its own that goes on where the login drops, " +
			"then claude's login, and ws's signing key on GitHub. " +
			"Either wizard follows the other's unit, and after the other's run, with nothing changed, runs no apply. " +
			"At a terminal the stages open as ws's list, and the ones chosen run: e opens ws server configure's forms. " +
			"Under --check each stage prints what it would do and changes nothing.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error { return runServerWizard(c, p, o) },
	}
	c.Flags().BoolVar(&o.check, "check", false, "print what each stage would do and change nothing")
	return c
}

// runServerWizard runs ws server wizard's stages under a lock, and what stays by hand of ws after a
// run in which a stage acted. It runs as root, as the remote script does: sudo asks its password
// before ws starts.
func runServerWizard(c *cobra.Command, p process, o wizardOptions) error {
	if p.euid != 0 {
		return errors.New("run it as root, by sudo: ws's values and its run are root's")
	}
	return wizardRun{c, p, serverSide(p), o}.start()
}
