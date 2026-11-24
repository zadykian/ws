//go:build darwin

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/local"
	macoscli "github.com/zadykian/ws/macos/cli"
)

// groups returns the command groups that exist on this system: ws macos, the Mac's ws unit and ws
// update, on a Mac. ws update ends in ws macos setup, and runs as the user, which asks sudo for the
// write. The Mac reads root's paths as the user, so that mise's plan, a unit's check and a run
// that changes nothing ask no password. ws's files in /etc/ws are readable by all.
func groups(p cli.Env) []*cobra.Command {
	p.Machine = local.OSMachine{UserReads: true}
	return []*cobra.Command{macosCmd(p), macoscli.UnitCmd(p), cli.Update(p, false, cli.FixedSetup("macos", "setup"))}
}
