package main

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/effect"
)

// newMacSecretsCmd returns ws mac secrets, which edits ws's secrets file: each command runs its own
// of ws server secrets on ws as root, over the wizard's ssh, with a value on ssh's stdin.
func newMacSecretsCmd(p process) *cobra.Command {
	c := group("secrets", "edit ws's secrets file, over ssh", append([]*cobra.Command{{Use: "init",
		Short: "make ws's age identity and an empty secrets file on ws", Args: cobra.NoArgs}},
		secretsEditCmds(p, nil)[1:]...)...)
	forward(c, p)
	return c
}

// forward has each command under c, but a group, run its own on ws.
func forward(c *cobra.Command, p process) {
	for _, sub := range c.Commands() {
		if sub.HasSubCommands() {
			forward(sub, p)
			continue
		}
		sub.RunE = func(c *cobra.Command, args []string) error { return runFar(c, p, args) }
	}
}

// runFar runs ws server secrets on ws with the command's path past ws mac secrets, and its arguments
// past a --, as a name may start with a dash: unset env A runs ws server secrets unset -- env A.
func runFar(c *cobra.Command, p process, args []string) error {
	if why := wizardRefusal(p); why != "" {
		return errors.New(why)
	}
	r := &varsReach{p: p}
	defer r.close(c.Context())
	ws, err := r.reachOn(c.Context(), p.machine, effect.NewTerminal(c.InOrStdin(), c.ErrOrStderr()), nil)
	if err != nil {
		return err
	}
	far := strings.Fields(c.CommandPath())[3:]
	if len(args) > 0 {
		far = append(append(far, "--"), args...)
	}
	out, err := ws.Secrets(c.Context(), nil, far...)
	_, _ = c.OutOrStdout().Write(out)
	return err
}
