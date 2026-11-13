// Package cli is the Mac's side of ws: its Target and its commands, under ws client macos.
package cli

import (
	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
)

// Command returns ws client macos, the commands that set up a Mac.
func Command(p enginecli.Env) *cobra.Command {
	return enginecli.Group("macos", "the commands for a Mac",
		newConfCmd(p), newConfigureCmd(p), newMacApplyCmd(p), newMacStateCmd(p), newWizardCmd(p), newResolverCmd(p),
		newTunnelCmd(p), newMacSecretsCmd(p))
}
