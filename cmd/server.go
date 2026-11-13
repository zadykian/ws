package cmd

import (
	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
	ubuntucli "github.com/zadykian/ws/roles/server/ubuntu/cli"
)

// newServerCmd returns ws server, the group of the server's systems. It builds everywhere, so that
// the tests run anywhere. Only its entry in the groups is built for Linux alone.
func newServerCmd(p cli.Env) *cobra.Command {
	return cli.Group("server", "the server's systems", ubuntucli.Command(p))
}
