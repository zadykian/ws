package cli

import (
	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/internal/sshkey"
)

// newSSHKeygenCmd returns ws macos ssh keygen.
func newSSHKeygenCmd(p enginecli.Env) *cobra.Command {
	c := &cobra.Command{
		Use:   "keygen NAME",
		Short: "make the key NAME in ~/.ssh by seward, or its .pub where the key alone is there",
		Long: "It checks NAME by ws's rule for a key's name, then runs seward keygen, which makes ~/.ssh/NAME and " +
			"NAME.pub. --store gives seward's store: enclave, its default, keeps the key in the Secure Enclave, " +
			"and file keeps it in the file, as on a virtual machine. seward's exit status is the command's.",
		Args: sshNameArg,
	}
	store := c.Flags().String("store", "", "seward's store for the key, enclave or file: seward's own default where left out")
	c.RunE = func(c *cobra.Command, args []string) error {
		argv := []string{"keygen", args[0]}
		if c.Flags().Changed("store") {
			argv = append(argv, "--store", *store)
		}
		return execSeward(p, argv)
	}
	return c
}

// sshNameArg checks that a command takes one name that internal/sshkey allows.
func sshNameArg(c *cobra.Command, args []string) error {
	if err := cobra.ExactArgs(1)(c, args); err != nil {
		return err
	}
	return sshkey.Check(args[0])
}
