//go:build linux

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/local"
)

// groups returns the command groups that exist on this system: ws server and ws unit, on Linux.
// It runs as root, which ws unit write checks, so the machine's root commands go without sudo: a
// container has none.
func groups(p cli.Env) []*cobra.Command {
	p.Machine = local.OSMachine{Sudo: []string{}}
	return []*cobra.Command{serverCmd(p), unitCmd(p)}
}
