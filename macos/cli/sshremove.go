package cli

import (
	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
)

// newSSHRemoveCmd returns ws macos ssh remove.
func newSSHRemoveCmd(p enginecli.Env) *cobra.Command {
	c := &cobra.Command{
		Use:   "remove NAME",
		Short: "delete the key NAME and its .pub by seward, which asks first",
		Long: "It checks NAME by ws's rule, then runs seward remove, which asks on the terminal before it " +
			"deletes ~/.ssh/NAME and NAME.pub. --yes skips the question. With no terminal and no --yes, seward " +
			"refuses. seward's exit status is the command's.",
		Args: sshNameArg,
	}
	yes := c.Flags().Bool("yes", false, "delete without asking")
	c.RunE = func(_ *cobra.Command, args []string) error {
		argv := []string{"remove", args[0]}
		if *yes {
			argv = append(argv, "--yes")
		}
		return execSeward(p, argv)
	}
	return c
}
