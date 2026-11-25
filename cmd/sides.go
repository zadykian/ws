package cmd

import (
	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
	macoscli "github.com/zadykian/ws/macos/cli"
	ubuntucli "github.com/zadykian/ws/ubuntu/cli"
)

// macosCmd returns ws macos, the Mac's commands. It builds everywhere, so that the tests run on
// Linux. Only its entry in the groups is built for darwin alone.
func macosCmd(p cli.Env) *cobra.Command { return macoscli.Command(p) }

// ubuntuCmd returns ws ubuntu, Ubuntu's commands. It builds everywhere, so that the
// tests run anywhere. Only its entry in the groups is built for Linux alone.
func ubuntuCmd(p cli.Env) *cobra.Command { return ubuntucli.Command(p) }
