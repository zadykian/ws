package cli

import (
	"io"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/roles/client/values"
)

// newValuesCmd returns ws client values.
func newValuesCmd(p enginecli.Env, server *string) *cobra.Command {
	var lines bool
	c := &cobra.Command{
		Use:   "print [FILE]",
		Short: "print the checked settings of values.yml and a server's file as shell assignments",
		Long: "It reads FILE, else $WS_VALUES, else /etc/ws/values.yml, and the server's file, " +
			"servers/NAME/values.yml beside it, of the server that --server names, and prints the checked settings as shell assignments, " +
			"or the errors of a file it refuses.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error { return runValuesPrint(c, p, server, args, lines) },
	}
	c.Flags().BoolVar(&lines, "lines", false, "check each line alone")
	return enginecli.Group("values", "check and print the Mac's values", c, newValuesCheckCmd(p, server))
}

// runValuesPrint prints the settings of FILE, or of the file that the environment names.
func runValuesPrint(c *cobra.Command, p enginecli.Env, server *string, args []string, lines bool) error {
	file := values.Path(p.Getenv, p.Pwd)
	if len(args) == 1 {
		file = args[0]
	}
	if !lines {
		name, err := values.Pick(p.Machine, file, *server)
		if err != nil {
			return err
		}
		settings, err := values.Load(p.Machine, file, values.ServerPath(file, name))
		if err != nil {
			return err
		}
		_, err = io.WriteString(c.OutOrStdout(), settings.Assignments())
		return err
	}
	name, err := values.Pick(p.Machine, file, *server)
	if err != nil {
		return err
	}
	settings, err := values.LoadLines(p.Machine, file, values.ServerPath(file, name))
	if err != nil {
		return err
	}
	_, err = io.WriteString(c.OutOrStdout(), settings.Assignments())
	return err
}
