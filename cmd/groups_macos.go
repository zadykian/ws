//go:build darwin

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/local"
	macoscli "github.com/zadykian/ws/roles/client/macos/cli"
)

// groups returns the command groups that exist on this system: ws client macos and the Mac's ws
// unit, on a Mac. ws unit reads root's paths as the user, so that mise's plan and a run that
// changes nothing ask no password.
func groups(p cli.Env) []*cobra.Command {
	p.Machine = local.OSMachine{}
	units := p
	units.Machine = local.OSMachine{UserReads: true}
	return []*cobra.Command{clientCmd(p), macoscli.UnitCmd(units)}
}
