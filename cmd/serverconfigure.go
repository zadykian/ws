package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/tui"
)

// newServerConfigureCmd returns ws server ubuntu configure.
func newServerConfigureCmd(p cli.Env) *cobra.Command {
	return &cobra.Command{
		Use:   "configure",
		Short: "edit ws's secrets and host_vars on forms, on ws",
		Long: "The forms of ws client macos configure that edit ws, read and written here as root, with no ssh. ws's " +
			"secrets show as set or not, never their values. Ctrl-N goes to host_vars. Ctrl-S writes a form. " +
			"It needs a terminal: anywhere else, run ws server ubuntu secrets and ws server ubuntu values set.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error { return runServerConfigure(c, p) },
	}
}

// runServerConfigure draws the forms. A form has no lines to fall back on, so it refuses where no
// screen draws.
func runServerConfigure(c *cobra.Command, p cli.Env) error {
	if p.Euid != 0 {
		return errors.New("run it as root, by sudo: ws's secrets and host_vars are root's")
	}
	if !drawsScreen(c, p) {
		return errors.New("it needs a terminal and no --plain: anywhere else, run ws server ubuntu secrets " +
			"and ws server ubuntu values set")
	}
	return tui.Configure(c.Context(), screenConfig(c, p), c.CommandPath(), serverSheets(p)...)
}

// serverSheets are the forms of ws server ubuntu configure and of ws server ubuntu wizard's list.
func serverSheets(p cli.Env) []tui.Sheet {
	return []tui.Sheet{serverSecretsSheet(p), hostVarsSheet(serverOpen)}
}

// serverSecretsSheet is ws's secrets file, edited here as ws server ubuntu secrets edits it.
func serverSecretsSheet(p cli.Env) tui.Sheet {
	return tui.Sheet{
		Name:   "secrets",
		Fields: func(map[string]string) []tui.FieldSpec { return nil },
		Open: func(ctx context.Context, b *tui.Bridge) (tui.Sheet, error) {
			on := p
			on.Machine = b.Effects().M
			store, err := serverSecretsStore(on)
			if err != nil {
				return tui.Sheet{}, err
			}
			held, err := store.Keys(ctx)
			if err != nil {
				return tui.Sheet{}, fmt.Errorf("the secrets on ws: %w", err)
			}
			return secretsSheet(store, held), nil
		},
	}
}
