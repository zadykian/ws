package cli

import (
	"context"

	"github.com/zadykian/ws/engine/tui"
	ubuntucli "github.com/zadykian/ws/roles/server/ubuntu/cli"
)

// secretsSheet is ws's secrets sheet before its read, which reaches ws as the host_vars sheet does.
func (r *varsReach) secretsSheet() tui.Sheet {
	return tui.Sheet{
		Name:   "secrets",
		Fields: func(map[string]string) []tui.FieldSpec { return nil },
		Open: func(ctx context.Context, b *tui.Bridge) (tui.Sheet, error) {
			ws, err := r.reach(ctx, b)
			if err != nil {
				return tui.Sheet{}, err
			}
			side := ubuntucli.WsSide{Ws: ws}
			held, err := side.Keys(ctx)
			if err != nil {
				return tui.Sheet{}, err
			}
			return ubuntucli.SecretsSheet(side, held), nil
		},
	}
}
