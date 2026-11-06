package main

import (
	"github.com/spf13/cobra"
)

// newMacCmd returns ws mac, the Mac's side of ws. It is built everywhere, so that the tests run on
// Linux. Only its entry in the groups is built for darwin alone.
func newMacCmd(p process) *cobra.Command {
	return group("mac", "the Mac's commands",
		newConfCmd(p), newMacApplyCmd(p), newMacStateCmd(p), newWizardCmd(p), newResolverCmd(p), newTunnelCmd(p), newSecretsCmd(p))
}
