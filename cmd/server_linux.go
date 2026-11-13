//go:build linux

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/core/effect/local"
)

// groups returns the command groups that exist on this system: ws server, on Linux. It runs as
// root, which ws server ubuntu apply checks, so the machine's root commands go without sudo: a
// container has none.
func groups(p process) []*cobra.Command {
	p.machine = local.OSMachine{Sudo: []string{}}
	return []*cobra.Command{newServerCmd(p)}
}
