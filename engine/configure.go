package engine

import (
	"context"
	"errors"

	"github.com/zadykian/ws/engine/tui"
)

// Configure draws a Target's forms. A form has no lines to fall back on, so it refuses where no
// screen draws.
func Configure(ctx context.Context, c Call, t Target) error {
	if t.Refuse != nil {
		if why := t.Refuse(c); why != "" {
			return errors.New(why)
		}
	}
	if !DrawsScreen(c) {
		return errors.New(t.NoScreen)
	}
	sheets, done, err := t.Sheets(ctx, c)
	if err != nil {
		return err
	}
	if done != nil {
		defer done(ctx)
	}
	return tui.Configure(ctx, ScreenConfig(c), c.Path, sheets...)
}
