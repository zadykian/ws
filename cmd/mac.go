package cmd

import (
	"github.com/spf13/cobra"
)

// newMacCmd returns ws client, the group of the client's systems. It builds everywhere, so that the
// tests run on Linux. Only its entry in the groups is built for darwin alone.
func newMacCmd(p process) *cobra.Command {
	return group("client", "the client's systems", newMacosCmd(p))
}

// newMacosCmd returns ws client macos, the commands that set up a Mac.
func newMacosCmd(p process) *cobra.Command {
	return group("macos", "the commands for a Mac",
		newConfCmd(p), newConfigureCmd(p), newMacApplyCmd(p), newMacStateCmd(p), newWizardCmd(p), newResolverCmd(p),
		newTunnelCmd(p), newMacSecretsCmd(p))
}
