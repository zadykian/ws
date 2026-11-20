package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine"
	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/roles/server/ubuntu/git"
	"github.com/zadykian/ws/roles/server/values"
)

// newGitCmd returns ws server ubuntu git: the procedures that the git role's units run.
func newGitCmd(p enginecli.Env) *cobra.Command {
	return enginecli.Group("git", "the git role's checks and procedures, which its units run",
		newGitCheckCmd(p), newGitIncludedCmd(p), newGitTokensCmd(p), newGitClonesCmd(p),
		newGitCredentialCmd(p))
}

// gitInputs reads what git's commands act on: the values and WS_USER's account. The tokens' file
// is the role's own to read.
func gitInputs(ctx context.Context, m effect.Machine) (values.Values, account.Account, error) {
	v, err := values.Load(m, serverVars)
	if err != nil {
		return v, account.Account{}, err
	}
	a, err := account.User{Name: v.User, Home: v.Home}.Lookup(ctx, m)
	return v, a, err
}

// newGitCheckCmd returns ws server ubuntu git check.
func newGitCheckCmd(p enginecli.Env) *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "refuse git's clones and tokens that the role would act on, and note what shadows ws's settings",
		Long: "It reads the values of " + serverVars + " and " + git.TokensFile + ", where there is one, or " + git.TokensVar +
			" in the secrets task's run, which writes the file. It fails where an " +
			"entry of WS_GIT_CLONES holds a user name or a token in its URL, or where the tokens' file is refused " +
			"(roles/server/ubuntu/git/README.md#git-credentials). No message quotes a token or a repository.\n\nIt prints a warning on stdout for each setting of ~/.gitconfig that shadows " +
			"/etc/ws/gitconfig's, by its name, never its value. The git role's guard runs it, before the tokens " +
			"and the clones.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			v, a, err := gitInputs(c.Context(), p.Machine)
			if err != nil {
				return err
			}
			notes, err := git.Check(c.Context(), p.Machine, a, v, p.TaskEnv(secretsTask))
			for _, n := range notes {
				_, _ = fmt.Fprintln(c.OutOrStdout(), "warning: "+n)
			}
			return err
		},
	}
}

// newGitIncludedCmd returns ws server ubuntu git included.
func newGitIncludedCmd(p enginecli.Env) *cobra.Command {
	return &cobra.Command{
		Use:   "included",
		Short: "say by the exit status whether the git config on stdin includes /etc/ws/gitconfig at its top",
		Long: "It reads the text of ~/.gitconfig on its standard input, as git does. It exits 0 where the text " +
			"includes /etc/ws/gitconfig once, at its top, and 1 where it lacks the include. It exits 2 with the " +
			"reason where the include sits below the file's own settings or comes twice, which it would then " +
			"override, or where git cannot read the text. The git role's unit of the include runs it, and " +
			"refuses the file on 2.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			text, err := io.ReadAll(c.InOrStdin())
			if err != nil {
				return engine.ExitStatus{Code: 2, Err: err}
			}
			holds, err := git.Included(c.Context(), p.Machine, "~/.gitconfig", text)
			switch {
			case err != nil:
				return engine.ExitStatus{Code: 2, Err: err}
			case !holds:
				return engine.ExitStatus{Code: 1}
			}
			return nil
		},
	}
}

// newGitCredentialCmd returns ws server ubuntu git credential.
func newGitCredentialCmd(p enginecli.Env) *cobra.Command {
	return &cobra.Command{
		Use:   "credential KEY [ACTION]",
		Short: "answer git's request for the token of a server besides GitHub, from " + git.TokensFile,
		Long: "git runs it as the account, as the credential program that /etc/ws/tokens.gitconfig gives " +
			"each URL, with the URL's key and git's action, and git's request on its standard input. For get, it prints the username and the " +
			"token of the URL whose key is KEY. For store and erase, or a key that the file lacks, it prints nothing, " +
			"so that git asks its next program. It reads no values file " +
			"(roles/server/ubuntu/git/README.md#git-credentials).",
		Args: cobra.RangeArgs(1, 2),
		RunE: func(c *cobra.Command, args []string) error {
			action := ""
			if len(args) == 2 {
				action = args[1]
			}
			return git.Answer(p.Machine, args[0], action, c.InOrStdin(), c.OutOrStdout())
		},
	}
}

// gitProcedure returns a command of git's that settles the resources of rs, with --check.
func gitProcedure(p enginecli.Env, use, short, long string, rs func(values.Values, account.Account) resource.Resource) *cobra.Command {
	return procedure(use, short, long, func(ctx context.Context, _ effect.Out) ([]resource.Resource, error) {
		v, a, err := gitInputs(ctx, p.Machine)
		if err != nil {
			return nil, err
		}
		return []resource.Resource{rs(v, a)}, nil
	})
}

// newGitTokensCmd returns ws server ubuntu git tokens.
func newGitTokensCmd(p enginecli.Env) *cobra.Command {
	return gitProcedure(p, "tokens", "write the credential programs of the URLs of "+git.TokensFile,
		"Each URL of "+git.TokensFile+" gets ws's credential program in /etc/ws/tokens.gitconfig, root's with the "+
			"account's group, which names no token (roles/server/ubuntu/git/README.md#git-credentials). Without the "+
			"file, tokens.gitconfig goes. In the secrets task's run, it reads "+git.TokensVar+", which that run writes "+
			"to the file, and a plan nowhere. The git role's run state runs it, after the role's check.",
		func(_ values.Values, a account.Account) resource.Resource {
			return git.Tokens{M: p.Machine, A: a, Lookup: p.TaskEnv(secretsTask)}
		})
}

// newGitClonesCmd returns ws server ubuntu git clones.
func newGitClonesCmd(p enginecli.Env) *cobra.Command {
	return gitProcedure(p, "clones", "clone each repository of WS_GIT_CLONES whose dest is missing, as WS_USER",
		"A clone is made once and never pulled (roles/server/ubuntu/git/README.md#clones). The git role's run "+
			"state runs it, after the tokens.",
		func(v values.Values, a account.Account) resource.Resource {
			return git.Clones{M: p.Machine, A: a, Clones: v.Git.Clones}
		})
}
