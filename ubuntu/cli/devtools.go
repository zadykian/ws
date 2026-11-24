package cli

import (
	"context"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/ubuntu/roles/devtools"
)

// newDevtoolsCmd returns ws ubuntu guest devtools: the procedures that the devtools role's units
// run.
func newDevtoolsCmd(p enginecli.Env) *cobra.Command {
	return enginecli.Group("devtools", "the devtools role's procedures, which its units run", newGHLoginCmd(p))
}

// newGHLoginCmd returns ws ubuntu guest devtools gh-login.
func newGHLoginCmd(p enginecli.Env) *cobra.Command {
	return procedure("gh-login", "log gh in to github.com as root, by the token of "+devtools.TokenFile,
		"gh logs in as root by the last line of "+devtools.TokenFile+", which it takes on its input "+
			"(ubuntu/roles/devtools/README.md#gh). In the secrets task's run, it reads "+devtools.TokenVar+
			", which that run writes to the file, and a plan nowhere. Without a token, gh's login stays. The "+
			"devtools role's run state runs it.",
		func(_ context.Context, _ effect.Out) ([]resource.Resource, error) {
			token, err := devtools.RunToken(p.Machine, p.TaskEnv(secretsTask))
			if err != nil {
				return nil, err
			}
			return []resource.Resource{devtools.GHLogin{M: p.Machine, U: rootUser,
				Token: token, Hidden: secretInputs}}, nil
		})
}
