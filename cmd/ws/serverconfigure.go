package main

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/internal/tui"
)

// serverSecretsSheet is ws's secrets file, edited here as ws server secrets edits it.
func serverSecretsSheet(p process) tui.Sheet {
	return tui.Sheet{
		Name:   "secrets",
		Fields: func(map[string]string) []tui.FieldSpec { return nil },
		Open: func(ctx context.Context, b *tui.Bridge) (tui.Sheet, error) {
			on := p
			on.machine = screenMachine(ctx, b, p.machine)
			store, err := serverSecretsStore(on)
			if err != nil {
				return tui.Sheet{}, err
			}
			held, err := store.keys(ctx)
			if err != nil {
				return tui.Sheet{}, fmt.Errorf("the secrets on ws: %w", err)
			}
			return secretsSheet(store, held), nil
		},
	}
}
