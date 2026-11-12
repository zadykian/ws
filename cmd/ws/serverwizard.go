package main

import (
	"context"
	"errors"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/wizard"
	"github.com/zadykian/ws/step"
)

// newServerWizardCmd returns ws server wizard.
func newServerWizardCmd(p process) *cobra.Command {
	var check bool
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
		RunE: func(c *cobra.Command, _ []string) error { return runServerWizard(c, p, check) },
	}
	c.Flags().BoolVar(&check, "check", false, "print what each stage would do and change nothing")
	return c
}

// runServerWizard runs ws server wizard's stages under a lock, and what stays by hand of ws after a
// run in which a stage acted. It runs as root, as the remote script does: sudo asks its password
// before ws starts.
func runServerWizard(c *cobra.Command, p process, check bool) error {
	if p.euid != 0 {
		return errors.New("run it as root, by sudo: ws's values and its run are root's")
	}
	if drawsScreen(c, p) {
		return serverScreen{c, p}.draw(check)
	}
	// The run's context ends with a signal, and its lock goes with it.
	ctx, stop := signal.NotifyContext(c.Context(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	term := effect.NewTerminal(c.InOrStdin(), c.ErrOrStderr())
	out := progOut{term, c.CommandPath(), map[string]bool{}}
	talks, _ := p.machine.(effect.Foreground)
	hand := &wizard.Hand{}
	stages := serverWizardStages(wizardEffects{p: p, ask: signalPrompt{term}, talks: talks, out: out, hand: hand},
		wizardOptions{check: check})
	closing := func(ctx context.Context) { hand.Close(ctx, c.OutOrStdout()); term.Flush() }
	_, err := wizard.Run(ctx, p.machine, step.Options{Observer: out, Check: check}, stages, p.lockDir(), 0, closing)
	return unwrapStep(err)
}
