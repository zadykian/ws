package main

import (
	"context"

	"github.com/zadykian/ws/internal/tui"
	"github.com/zadykian/ws/internal/wizard"
)

// macState is the state screen's read of side's machine, and what ends it. ws's reach takes sudo, the
// scan's, which the run takes after it, and lets the lock go with the screen.
func macState(p process, side tui.Side, sudo *wizard.Sudo) (tui.StateReader, func(context.Context)) {
	if side == tui.MacSide {
		return macRead(p), nil
	}
	reach := &varsReach{p: p, sudo: sudo}
	return stateRead(reach.reach, p.version), reach.close
}

// stateRead is the state screen's read of ws, by the remote script's report, which open reaches ws
// for. The session stays once found, as a new one would hold sudo's ticket until the screen ends.
// Each read asks ws's status again, as a run may have changed it.
func stateRead(open sessionOpen, version string) tui.StateReader {
	var ws *wizard.Session
	return func(ctx context.Context, b *tui.Bridge) ([]tui.StateSection, error) {
		if ws == nil {
			found, err := open(ctx, b)
			if err != nil {
				return nil, err
			}
			ws = found
		}
		states, err := ws.Restate(ctx, version)
		return roleSections(states), err
	}
}
