package main

import (
	"context"

	"github.com/zadykian/ws/internal/tui"
	"github.com/zadykian/ws/internal/wizard"
	"github.com/zadykian/ws/mac"
)

// stated draws the state screen that choice names, then the lists as it left them.
func (w macScreen) stated(choice tui.Choice, sudo *wizard.Sudo) (tui.Choice, error) {
	if err := w.state(choice.Side, sudo); err != nil {
		return tui.Choice{}, err
	}
	return tui.Reopen(w.c.Context(), w.config(), choice)
}

// state draws the state screen of side's machine. sudo is the scan's, which the run takes after it.
func (w macScreen) state(side tui.Side, sudo *wizard.Sudo) error {
	ctx, title := w.c.Context(), w.c.CommandPath()
	if side == tui.MacSide {
		return tui.RunState(ctx, w.config(), title+": Mac", macRead(w.p))
	}
	reach := &varsReach{p: w.p, sudo: sudo}
	defer reach.close(ctx)
	return tui.RunState(ctx, w.config(), title+": ws", wsRead(reach, w.p.version))
}

// wsRead is the state screen's read of ws. The way to ws stays once found, as a new one would hold
// sudo's ticket until the screen ends. Each read is a Remote of its own, which asks ws's status
// again, as a run may have changed it.
func wsRead(reach *varsReach, version string) tui.StateReader {
	var shell wizard.Shell
	return func(ctx context.Context, b *tui.Bridge) ([]tui.StateSection, error) {
		if shell == nil {
			remote, err := reach.reach(ctx, b, mac.RemoteScript)
			if err != nil {
				return nil, err
			}
			shell = remote.Shell
		}
		ws := &wizard.Remote{Shell: shell, Script: mac.RemoteScript}
		states, err := ws.State(ctx, version)
		return roleSections(states), err
	}
}
