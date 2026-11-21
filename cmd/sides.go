package cmd

import (
	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
	macoscli "github.com/zadykian/ws/roles/client/cli"
	ubuntucli "github.com/zadykian/ws/roles/server/cli"
)

// clientCmd returns ws client, the Mac's commands. It builds everywhere, so that the tests run on
// Linux. Only its entry in the groups is built for darwin alone.
func clientCmd(p cli.Env) *cobra.Command { return macoscli.Command(p) }

// serverCmd returns ws server, the Ubuntu server's commands. It builds everywhere, so that the
// tests run anywhere. Only its entry in the groups is built for Linux alone.
func serverCmd(p cli.Env) *cobra.Command { return ubuntucli.Command(p) }
