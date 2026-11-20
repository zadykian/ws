// Package cli is the server's side of ws: its commands under ws server ubuntu, which its roles'
// units and the maintainer run.
package cli

import (
	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/units"
	"github.com/zadykian/ws/roles/server/ubuntu/ikeward"
)

const serverVars = "/etc/ws/values.yml"

// Command returns ws server ubuntu, the commands that an Ubuntu server's units and its maintainer run.
func Command(p enginecli.Env) *cobra.Command {
	return enginecli.Group("ubuntu", "the commands for an Ubuntu server", newFansCmd(p),
		newCableCmd(p, units.OSUnits{}), newTunnelCmd(p, units.OSUnits{}), newIkewardCmd(p, ikeward.StatusTries, ikeward.StatusWait),
		newGitCmd(p), newDevtoolsCmd(p), newClaudeCmd(p), newValuesCmd(p), newSignozCmd(p))
}
