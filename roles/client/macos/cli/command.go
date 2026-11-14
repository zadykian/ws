// Package cli is the Mac's side of ws: its Target and its commands, under ws client macos.
package cli

import (
	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
)

// Command returns ws client macos, the commands that set up a Mac.
func Command(p enginecli.Env) *cobra.Command {
	var server string
	g := enginecli.Group("macos", "the commands for a Mac",
		newConfCmd(p, &server), newConfigureCmd(p, &server), newMacApplyCmd(p, &server), newMacStateCmd(p, &server), newWizardCmd(p, &server),
		newResolverCmd(p), newTunnelCmd(p, &server), newMacSecretsCmd(p, &server),
		newServersCmd(p, &server))
	g.PersistentFlags().StringVar(&server, "server", "", "the server to act on, which may be left out where there is one")
	return g
}
