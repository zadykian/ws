//go:build !darwin && !linux

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
)

// groups returns the command groups that exist on this system: none here, as ws client is built for
// darwin alone and ws server for Linux alone.
func groups(cli.Env) []*cobra.Command { return nil }
