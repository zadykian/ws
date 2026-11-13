package cli

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/stages/host"
	ubuntucli "github.com/zadykian/ws/roles/server/ubuntu/cli"
)

// newConfigureCmd returns ws client macos configure.
func newConfigureCmd(p enginecli.Env) *cobra.Command {
	return &cobra.Command{
		Use:   "configure",
		Short: "edit mac.conf, the secrets and ws's host_vars on forms",
		Long: "It shows the keys that ws client macos wizard would ask, by WS_TARGET, with what mac.conf holds, and " +
			"writes them on Ctrl-S. Ctrl-N goes to ws's secrets, which show as set or not, never their values, " +
			"then to ws's host_vars, both read over ssh. It needs a terminal: anywhere else, edit the file by hand.",
		Args: cobra.NoArgs,
		RunE: enginecli.Configure(macTarget(p, new([]string)), p),
	}
}

// macForms gives the forms' sheets, for ws client macos configure and the wizard's list, and done,
// which lets go of ws once they have ended.
func macForms(in io.Reader, errw io.Writer, p enginecli.Env) func(context.Context) ([]tui.Sheet, func(context.Context), error) {
	return func(context.Context) ([]tui.Sheet, func(context.Context), error) {
		form := conf.Form{Conf: confPath(p.Getenv, p.Pwd), Defaults: shellDefaults(), M: p.Machine}
		form, err := form.Load(effect.NewTerminal(in, errw))
		if err != nil {
			return nil, nil, err
		}
		// One way to root serves both of ws's sheets, so that sudo's password is asked once.
		vars := &varsReach{p: p, sudo: &host.Sudo{}}
		return []tui.Sheet{confSheet(form), vars.secretsSheet(), ubuntucli.HostVarsSheet(vars.reach)}, vars.close, nil
	}
}

// confSheet is mac.conf on the Configure screen.
func confSheet(form conf.Form) tui.Sheet {
	return tui.Sheet{
		Name:   "mac.conf",
		Fields: func(edits map[string]string) []tui.FieldSpec { return enginecli.FieldSpecs(form.Fields(edits)) },
		Save: func(ctx context.Context, b *tui.Bridge, edits map[string]string) (tui.Sheet, error) {
			// A copy, as Fields reads form on the screen's goroutine.
			work := form
			work.M = b.Effects().M
			next, err := work.Save(ctx, b, edits)
			return confSheet(next), err
		},
	}
}
