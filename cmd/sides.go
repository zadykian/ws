package cmd

import (
	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
	macoscli "github.com/zadykian/ws/roles/client/macos/cli"
	ubuntucli "github.com/zadykian/ws/roles/server/ubuntu/cli"
)

// clientCmd returns ws client, the group of the client's systems. It builds everywhere, so that the
// tests run on Linux. Only its entry in the groups is built for darwin alone.
func clientCmd(p cli.Env) *cobra.Command {
	return cli.Group("client", "the client's systems", macoscli.Command(p))
}

// serverCmd returns ws server, the group of the server's systems. It builds everywhere, so that
// the tests run anywhere. Only its entry in the groups is built for Linux alone.
func serverCmd(p cli.Env) *cobra.Command {
	return cli.Group("server", "the server's systems", ubuntucli.Command(p))
}
