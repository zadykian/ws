package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
)

// sewardPath is where ssh's unit copies seward, the tool that owns the Mac's keys.
const sewardPath = "/usr/local/bin/seward"

// newSSHCmd returns ws macos ssh: the commands of ssh's units, and the keys' commands, which hand
// seward the call.
func newSSHCmd(p enginecli.Env) *cobra.Command {
	return enginecli.Group("ssh", "the commands of ssh's units and of the Mac's keys",
		newSSHIncludedCmd(), newSSHListCmd(p), newSSHKeygenCmd(p), newSSHRemoveCmd(p))
}

// newSSHListCmd returns ws macos ssh list.
func newSSHListCmd(p enginecli.Env) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "list the keys in ~/.ssh by seward: a line each, with its type, fingerprint and store",
		Long: "It runs seward list, which prints a line for each key: its NAME, type, fingerprint and store, " +
			"tab-separated. seward's exit status is the command's.",
		Args: cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error { return execSeward(p, []string{"list"}) },
	}
}

// execSeward replaces ws with seward and argv, which hands it the terminal and its exit status. It
// returns only where seward cannot start. seward's keys and agent are the user's, so root is refused.
func execSeward(p enginecli.Env, argv []string) error {
	if p.Euid == 0 {
		return errors.New("run it as your user, not root: seward's keys and its agent are the user's")
	}
	if err := p.Machine.Exec(sewardPath, argv, nil); err != nil {
		return fmt.Errorf("cannot run %s (%w): ws macos setup installs seward", sewardPath, err)
	}
	return nil
}
