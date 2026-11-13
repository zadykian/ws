package cmd

import (
	"runtime/debug"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
)

// versionOf returns the version that Go stamped into the build: the tag, or a pseudo-version with
// the commit and +dirty. It returns "devel" where there is no build info or Go stamped (devel).
func versionOf(info *debug.BuildInfo, ok bool) string {
	if !ok || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return "devel"
	}
	return info.Main.Version
}

// newVersionCmd returns ws version, which prints this binary's version.
func newVersionCmd(p cli.Env) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "print the version of this ws",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			_, err := c.OutOrStdout().Write([]byte(p.Version + "\n"))
			return err
		},
	}
}
