// Package cli is the Mac's side of ws: its commands under ws macos, which its roles' units
// and the user run.
package cli

import (
	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
)

// Command returns ws macos, the commands that a Mac's units and its user run.
func Command(p enginecli.Env) *cobra.Command {
	var server string
	g := enginecli.Group("macos", "the commands for a Mac",
		newValuesCmd(p, &server),
		newResolverCmd(p), newTunnelCmd(p, &server),
		newServersCmd(p, &server), newRemoveCmd(p, &server), newCldCmd(p), newKeysCmd(p), newCableCmd(p), newNetworksCmd(p), newSSHCmd(), newGitCmd(p),
		newSecretsCmd(p, &server), newSetupCmd(p, &server, liveSetup()))
	g.PersistentFlags().StringVar(&server, "server", "", "the server to act on, which may be left out where there is one")
	return g
}
