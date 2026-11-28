package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine"
	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/internal/sshcfg"
	"github.com/zadykian/ws/macos/roles/git"
)

// The has commands of the Mac's line units: each reads the user's file on stdin. It exits 0 where
// the file holds ws's line in its place, and 1 where it lacks it. It exits 2 with the reason where
// the line sits out of place, which the unit refuses.

// newSSHIncludedCmd returns ws macos ssh included.
func newSSHIncludedCmd() *cobra.Command {
	return hasCmd("included",
		"say by the exit status whether the ssh config on stdin includes /etc/ws/ssh_config at its top",
		"It exits 0 where the Include of /etc/ws/ssh_config comes before any setting, and 1 where the text "+
			"lacks it. It exits 2 where the Include sits below a setting, which ssh would take first. "+
			"ssh's unit of the Include runs it.",
		func(_ *cobra.Command, text []byte) (bool, error) {
			place, line := sshcfg.IncludeAt(string(text), sshcfg.File)
			if place == sshcfg.Below {
				return false, fmt.Errorf("line %d of ~/.ssh/config is the Include of %s, below other settings, "+
					"which ssh would take first: move it to the top", line, sshcfg.File)
			}
			return place == sshcfg.First, nil
		})
}

// newGitCmd returns ws macos git, with git's has command.
func newGitCmd(p enginecli.Env) *cobra.Command {
	return enginecli.Group("git", "the commands of git's units", hasCmd("included",
		"say by the exit status whether the git config on stdin includes /etc/ws/gitconfig at its top",
		"It reads the text as git does. It exits 0 where the text includes /etc/ws/gitconfig once, at its "+
			"top, and 1 where it lacks the include. It exits 2 where the include sits below the file's own "+
			"settings or comes twice, which it would then override, or where git cannot read the text. "+
			"git's unit of the include runs it.",
		func(c *cobra.Command, text []byte) (bool, error) { return git.Included(c.Context(), p.Machine, text) }))
}

// hasCmd returns a has command, which holds reads from the text on stdin.
func hasCmd(use, short, long string, holds func(*cobra.Command, []byte) (bool, error)) *cobra.Command {
	return &cobra.Command{Use: use, Short: short, Long: long, Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			text, err := io.ReadAll(c.InOrStdin())
			if err != nil {
				return engine.ExitStatus{Code: 2, Err: err}
			}
			ok, err := holds(c, text)
			switch {
			case err != nil:
				return engine.ExitStatus{Code: 2, Err: err}
			case !ok:
				return engine.ExitStatus{Code: 1}
			}
			return nil
		}}
}
