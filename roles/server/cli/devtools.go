package cli

import (
	"context"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/roles/server/ubuntu/devtools"
	"github.com/zadykian/ws/roles/server/values"
)

// newDevtoolsCmd returns ws server devtools: the procedures that the devtools role's units
// run.
func newDevtoolsCmd(p enginecli.Env) *cobra.Command {
	return enginecli.Group("devtools", "the devtools role's procedures, which its units run", newGHLoginCmd(p))
}

// newGHLoginCmd returns ws server devtools gh-login.
func newGHLoginCmd(p enginecli.Env) *cobra.Command {
	return procedure("gh-login", "log gh in to github.com as WS_USER, by the token of "+devtools.TokenFile,
		"gh logs in as WS_USER by the last line of "+devtools.TokenFile+", which it takes on its input "+
			"(roles/server/ubuntu/devtools/README.md#gh). In the secrets task's run, it reads "+devtools.TokenVar+
			", which that run writes to the file, and a plan nowhere. Without a token, gh's login stays. The "+
			"devtools role's run state runs it.",
		func(_ context.Context, _ effect.Out) ([]resource.Resource, error) {
			v, err := values.Load(p.Machine, serverVars)
			if err != nil {
				return nil, err
			}
			token, err := devtools.RunToken(p.Machine, p.TaskEnv(secretsTask))
			if err != nil {
				return nil, err
			}
			return []resource.Resource{devtools.GHLogin{M: p.Machine, U: account.User{Name: v.User, Home: v.Home},
				Token: token, Hidden: secretInputs}}, nil
		})
}
