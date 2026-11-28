// Package cli is Ubuntu's side of ws: its commands under ws ubuntu. The guest's are ws ubuntu
// guest, the host's ws ubuntu host, and both sides' ws ubuntu ssh. Its roles' units and the
// maintainer run them.
package cli

import (
	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/units"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/ubuntu/roles/ikeward"
)

// guestVars and hostVars are each side's values file, on the machine of that side alone.
const (
	guestVars = "/etc/ws/values.yml"
	hostVars  = "/etc/ws/values.yml"
)

// rootUser is the one user the commands act for.
var rootUser = account.User{Name: "root", Home: "/root"}

// Command returns ws ubuntu, the commands of Ubuntu's side. ws ubuntu guest is for a guest's units
// and its maintainer, and ws ubuntu host for a host's. ws ubuntu ssh serves both.
func Command(p enginecli.Env) *cobra.Command {
	return enginecli.Group("ubuntu", "the commands for an Ubuntu machine", guestCmd(p), hostCmd(p), newSSHCmd(p))
}

// hostCmd returns ws ubuntu host, the commands of a host: the machine that runs the guests.
func hostCmd(p enginecli.Env) *cobra.Command {
	return enginecli.Group("host", "the commands for an Ubuntu host", newFansCmd(p),
		newCableCmd(p, units.OSUnits{}), newIncusCmd(p), newGuestsCmd(p),
		newTunnelCmd(p, units.OSUnits{}, hostTunnel, newTunnelGuestsCmd(p, units.OSUnits{})), newValuesCmd(p, hostValues), newSignozCmd(p, hostSignoz),
		newSetupCmd(p, liveSetup(), hostSetup))
}

// guestCmd returns ws ubuntu guest, the commands of a guest: a container or a cloud VM that runs
// the roles.
func guestCmd(p enginecli.Env) *cobra.Command {
	return enginecli.Group("guest", "the commands for an Ubuntu guest", newTunnelCmd(p, units.OSUnits{}, guestTunnel),
		newIkewardCmd(p, ikeward.StatusTries, ikeward.StatusWait),
		newGitCmd(p), newDevtoolsCmd(p), newClaudeCmd(p, units.OSUnits{}), newValuesCmd(p, guestValues), newSignozCmd(p, guestSignoz),
		newSetupCmd(p, liveSetup(), guestSetup))
}
