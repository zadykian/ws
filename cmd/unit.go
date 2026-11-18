package cmd

import (
	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
	ubuntucli "github.com/zadykian/ws/roles/server/ubuntu/cli"
)

// unitCmd returns ws unit, with the server's side, as mise calls it at ws's root. It builds
// everywhere, so that the tests run anywhere. Only its entry in the groups is built for Linux
// alone. The Mac's ws unit is its own side's, which groups_macos.go builds.
func unitCmd(p cli.Env) *cobra.Command { return ubuntucli.UnitCmd(p) }
