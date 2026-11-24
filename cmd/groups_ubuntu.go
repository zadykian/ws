//go:build linux

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/local"
	ubuntucli "github.com/zadykian/ws/roles/server/cli"
)

// groups returns the command groups that exist on this system: ws server, ws unit and ws update, on
// Linux. ws update ends in the setup of the machine's side, and runs as root.
// It runs as root, which ws unit write checks, so the machine's root commands go without sudo: a
// container has none.
func groups(p cli.Env) []*cobra.Command {
	p.Machine = local.OSMachine{Sudo: []string{}}
	return []*cobra.Command{serverCmd(p), unitCmd(p), cli.Update(p, true, ubuntucli.UpdateSetups())}
}
