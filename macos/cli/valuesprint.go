package cli

import (
	"io"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/macos/values"
)

// newValuesCmd returns ws macos values.
func newValuesCmd(p enginecli.Env, server *string) *cobra.Command {
	var lines bool
	c := &cobra.Command{
		Use:   "print [DIR]",
		Short: "print the checked settings of a server's values.yml as shell assignments",
		Long: "It reads servers/NAME/values.yml in DIR, else in $WS_VALUES, else in /etc/ws, of the server that " +
			"--server names, and prints the checked settings as shell assignments, or the errors of a file it refuses.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error { return runValuesPrint(c, p, server, args, lines) },
	}
	c.Flags().BoolVar(&lines, "lines", false, "check each line alone")
	return enginecli.Group("values", "check and print the Mac's values", c, newValuesCheckCmd(p, server))
}

// runValuesPrint prints the settings of a server of DIR, or of the directory that the environment
// names.
func runValuesPrint(c *cobra.Command, p enginecli.Env, server *string, args []string, lines bool) error {
	dir := values.Dir(p.Getenv, p.Pwd)
	if len(args) == 1 {
		dir = args[0]
	}
	if !lines {
		name, err := values.Pick(p.Machine, dir, *server)
		if err != nil {
			return err
		}
		settings, err := values.ReadFleet(p.Machine, dir).Load(name)
		if err != nil {
			return err
		}
		_, err = io.WriteString(c.OutOrStdout(), settings.Assignments())
		return err
	}
	name, err := values.Pick(p.Machine, dir, *server)
	if err != nil {
		return err
	}
	settings, err := values.LoadLines(p.Machine, values.ServerPath(dir, name))
	if err != nil {
		return err
	}
	_, err = io.WriteString(c.OutOrStdout(), settings.Assignments())
	return err
}
