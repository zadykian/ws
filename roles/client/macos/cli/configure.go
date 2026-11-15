package cli

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/roles/client/macos/stages/host"
	"github.com/zadykian/ws/roles/client/macos/values"
	ubuntucli "github.com/zadykian/ws/roles/server/ubuntu/cli"
)

// newConfigureCmd returns ws client macos configure.
func newConfigureCmd(p enginecli.Env, server *string) *cobra.Command {
	return &cobra.Command{
		Use:   "configure",
		Short: "edit the Mac's values.yml, the server's file, the secrets and ws's values.yml on forms",
		Long: "It shows the keys that ws client macos wizard would ask, by WS_TARGET, with what the files hold: " +
			"the Mac's values.yml, then those of the server's file, servers/NAME/values.yml. It writes them on Ctrl-S. " +
			"Ctrl-N goes on to ws's secrets, which show as set or not, never their values, then to ws's values.yml, both read over ssh. It needs a terminal: anywhere else, edit the file by hand.",
		Args: cobra.NoArgs,
		RunE: enginecli.Configure(macTarget(p, new([]string), server), p),
	}
}

// macForms gives the forms' sheets, for ws client macos configure and the wizard's list, and done,
// which lets go of ws once they have ended.
func macForms(in io.Reader, errw io.Writer, p enginecli.Env, server *string) func(context.Context) ([]tui.Sheet, func(context.Context), error) {
	return func(context.Context) ([]tui.Sheet, func(context.Context), error) {
		name, err := pickServer(p, server)
		if err != nil {
			return nil, nil, err
		}
		macConf := values.Path(p.Getenv, p.Pwd)
		form := values.Form{Conf: macConf, Server: values.ServerPath(macConf, name), M: p.Machine}
		form, err = form.Load(effect.NewTerminal(in, errw))
		if err != nil {
			return nil, nil, err
		}
		macForm, serverForm := form, form
		macForm.Scope, serverForm.Scope = values.MacFile, values.ServerFile
		// One way to root serves both of ws's sheets, so that sudo's password is asked once.
		vars := &varsReach{p: p, server: name, sudo: &host.Sudo{}}
		return []tui.Sheet{confSheet("values.yml", macForm), confSheet("servers/"+name+"/values.yml", serverForm), vars.secretsSheet(), ubuntucli.ValuesSheet(vars.reach)}, vars.close, nil
	}
}

// confSheet is one of the two files on the Configure screen, the form's scope, under name.
func confSheet(name string, form values.Form) tui.Sheet {
	return tui.Sheet{
		Name:   name,
		Fields: func(edits map[string]string) []tui.FieldSpec { return enginecli.FieldSpecs(form.Fields(edits)) },
		Save: func(ctx context.Context, b *tui.Bridge, edits map[string]string) (tui.Sheet, error) {
			// A copy, as Fields reads form on the screen's goroutine.
			work := form
			work.M = b.Effects().M
			next, err := work.Save(ctx, b, edits)
			return confSheet(name, next), err
		},
	}
}
