package cmd

import (
	"context"
	"errors"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/form"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/stages/host"
)

// newConfigureCmd returns ws client macos configure.
func newConfigureCmd(p process) *cobra.Command {
	return &cobra.Command{
		Use:   "configure",
		Short: "edit mac.conf, the secrets and ws's host_vars on forms",
		Long: "It shows the keys that ws client macos wizard would ask, by WS_TARGET, with what mac.conf holds, and " +
			"writes them on Ctrl-S. Ctrl-N goes to ws's secrets, which show as set or not, never their values, " +
			"then to ws's host_vars, both read over ssh. It needs a terminal: anywhere else, edit the file by hand.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error { return runConfigure(c, p) },
	}
}

// runConfigure draws the form. A form has no lines to fall back on, so it refuses where no screen
// draws. A mac.conf that it can't show stops it on the terminal, before the screen.
func runConfigure(c *cobra.Command, p process) error {
	if why := wizardRefusal(p); why != "" {
		return errors.New(why)
	}
	if !drawsScreen(c, p) {
		return errors.New("it needs a terminal and no --plain: anywhere else, edit mac.conf by hand")
	}
	sheets, done, err := macForms(c, p)(c.Context())
	if err != nil {
		return err
	}
	defer done(c.Context())
	return tui.Configure(c.Context(), screenConfig(c, p), c.CommandPath(), sheets...)
}

// macForms gives the forms' sheets, for ws client macos configure and the wizard's list, and done,
// which lets go of ws once they have ended.
func macForms(c *cobra.Command, p process) func(context.Context) ([]tui.Sheet, func(context.Context), error) {
	return func(context.Context) ([]tui.Sheet, func(context.Context), error) {
		form := conf.Form{Conf: confPath(p.getenv, p.pwd), Defaults: shellDefaults(), M: p.machine}
		form, err := form.Load(effect.NewTerminal(c.InOrStdin(), c.ErrOrStderr()))
		if err != nil {
			return nil, nil, err
		}
		// One way to root serves both of ws's sheets, so that sudo's password is asked once.
		vars := &varsReach{p: p, sudo: &host.Sudo{}}
		return []tui.Sheet{confSheet(form), vars.secretsSheet(), hostVarsSheet(vars.reach)}, vars.close, nil
	}
}

// confSheet is mac.conf on the Configure screen.
func confSheet(form conf.Form) tui.Sheet {
	return tui.Sheet{
		Name:   "mac.conf",
		Fields: func(edits map[string]string) []tui.FieldSpec { return fieldSpecs(form.Fields(edits)) },
		Save: func(ctx context.Context, b *tui.Bridge, edits map[string]string) (tui.Sheet, error) {
			// A copy, as Fields reads form on the screen's goroutine.
			work := form
			work.M = b.Effects().M
			next, err := work.Save(ctx, b, edits)
			return confSheet(next), err
		},
	}
}

// fieldSpecs is a form's fields on the screen. A key that takes one of a few is a choice.
func fieldSpecs(fields []form.Field) []tui.FieldSpec {
	specs := make([]tui.FieldSpec, 0, len(fields))
	for _, f := range fields {
		spec := tui.FieldSpec{Label: f.Key, Help: f.Text, Value: f.Value, Default: f.Default,
			Options: f.Options, Valid: f.Valid}
		if f.Options != nil {
			spec.Kind = tui.ChoiceField
		}
		specs = append(specs, spec)
	}
	return specs
}
