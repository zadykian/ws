//go:build darwin

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/core/effect/local"
)

// groups returns the command groups that exist on this system: ws client macos, on a Mac.
func groups(p process) []*cobra.Command {
	p.machine = local.OSMachine{}
	return []*cobra.Command{newMacCmd(p)}
}
