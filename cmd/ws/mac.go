package main

import (
	"github.com/spf13/cobra"
)

// newMacCmd returns ws mac, the Mac's side of ws. It is built everywhere, so that the tests run on
// Linux. Only its entry in the groups is built for darwin alone.
func newMacCmd(p process) *cobra.Command {
	return group("mac", "the Mac's commands",
		newConfCmd(p), newMacApplyCmd(p), newWizardCmd(p), newResolverCmd(p), newTunnelCmd(p), newSecretsCmd(p))
}

// newSecretsCmd returns ws mac secrets. Until its commands are cobra's own, it passes the arguments
// to secretsCommand and the status back as an exitStatus.
func newSecretsCmd(p process) *cobra.Command {
	return &cobra.Command{
		Use:                "secrets",
		Short:              "keep the secrets file, with its identity in the Keychain",
		DisableFlagParsing: true,
		Args:               cobra.ArbitraryArgs,
		RunE: func(c *cobra.Command, args []string) error {
			if status := secretsCommand(c.Context(), args, p.machine, p.getenv, c.InOrStdin(), c.OutOrStdout(), c.ErrOrStderr()); status != 0 {
				return exitStatus(status)
			}
			return nil
		},
	}
}
