// Package cli is the server's side of ws: its commands under ws server, which its roles'
// units and the maintainer run.
package cli

import (
	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/units"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/roles/server/ubuntu/ikeward"
)

const serverVars = "/etc/ws/values.yml"

// rootUser is the one user the commands act for.
var rootUser = account.User{Name: "root", Home: "/root"}

// Command returns ws server, the commands that an Ubuntu server's units and its maintainer run.
func Command(p enginecli.Env) *cobra.Command {
	return enginecli.Group("server", "the commands for an Ubuntu server", newFansCmd(p),
		newCableCmd(p, units.OSUnits{}), newTunnelCmd(p, units.OSUnits{}), newIkewardCmd(p, ikeward.StatusTries, ikeward.StatusWait),
		newGitCmd(p), newDevtoolsCmd(p), newClaudeCmd(p, units.OSUnits{}), newValuesCmd(p), newSignozCmd(p),
		newSetupCmd(p, liveSetup()))
}
