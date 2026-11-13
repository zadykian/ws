package cmd

import (
	"context"
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
		RunE: cli.Configure(serverTarget(p, new(applyOptions)), p),
	}
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
