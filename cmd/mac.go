package cmd

import (
	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
	macoscli "github.com/zadykian/ws/roles/client/macos/cli"
)

// newMacCmd returns ws client, the group of the client's systems. It builds everywhere, so that the
// tests run on Linux. Only its entry in the groups is built for darwin alone.
func newMacCmd(p cli.Env) *cobra.Command {
	return cli.Group("client", "the client's systems", macoscli.Command(p))
}
