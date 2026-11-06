package main

import (
	"context"

	"github.com/zadykian/ws/internal/tui"
	"github.com/zadykian/ws/internal/wizard"
	"github.com/zadykian/ws/mac"
)

// wsRead is the state screen's read of ws. The way to ws stays once found, as a new one would hold
// sudo's ticket until the screen ends. Each read asks ws's status again, as a run may have changed it.
func wsRead(reach *varsReach, version string) tui.StateReader {
	var ws *wizard.Remote
	return func(ctx context.Context, b *tui.Bridge) ([]tui.StateSection, error) {
		if ws == nil {
			remote, err := reach.reach(ctx, b, mac.RemoteScript)
			if err != nil {
				return nil, err
			}
			ws = remote
		}
		ws.Forget()
		states, err := ws.State(ctx, version)
		return roleSections(states), err
	}
}
