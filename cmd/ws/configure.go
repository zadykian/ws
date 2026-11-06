package main

import (
	"context"
	"errors"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/tui"
	"github.com/zadykian/ws/internal/wizard"
)

// newConfigureCmd returns ws mac configure.
func newConfigureCmd(p process) *cobra.Command {
	return &cobra.Command{
		Use:   "configure",
		Short: "edit mac.conf and the secrets on forms",
		Long: "It shows the keys that ws mac wizard would ask, by WS_TARGET, with what mac.conf holds, and " +
			"writes them on Ctrl-S. Ctrl-N goes to the secrets, which show as set or not, never their values. It needs a terminal: anywhere else, edit the file by hand.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error { return runConfigure(c, p) },
	}
}

// runConfigure draws the form. A form has no lines to fall back on, so it refuses where no screen
// draws, and a mac.conf that it can't show stops it on the terminal, before the screen.
func runConfigure(c *cobra.Command, p process) error {
	if why := wizardRefusal(p); why != "" {
		return errors.New(why)
	}
	if !drawsScreen(c, p) {
		return errors.New("it needs a terminal and no --plain: anywhere else, edit mac.conf by hand")
	}
	form := wizard.ConfForm{Conf: confPath(p.getenv, p.pwd), Defaults: shellDefaults(), M: p.machine}
	form, err := form.Load(effect.NewTerminal(c.InOrStdin(), c.ErrOrStderr()))
	if err != nil {
		return err
	}
	store := newSecretsStore(p)
	held, err := heldSecrets(c.Context(), store)
	if err != nil {
		return err
	}
	return tui.Configure(c.Context(), screenConfig(c, p), c.CommandPath(), confSheet(form), secretsSheet(store, held))
}

// confSheet is mac.conf on the Configure screen.
func confSheet(form wizard.ConfForm) tui.Sheet {
	return tui.Sheet{
		Name:   "mac.conf",
		Fields: func(edits map[string]string) []tui.FieldSpec { return fieldSpecs(form.Fields(edits)) },
		Save: func(ctx context.Context, b *tui.Bridge, edits map[string]string) (tui.Sheet, error) {
			// A copy, as Fields reads form on the screen's goroutine.
			work := form
			work.M = screenMachine(ctx, b, form.M)
			next, err := work.Save(ctx, b, edits)
			return confSheet(next), err
		},
	}
}

// fieldSpecs is a form's fields on the screen. A key that takes one of a few is a choice.
func fieldSpecs(fields []wizard.Field) []tui.FieldSpec {
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
