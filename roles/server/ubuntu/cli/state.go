package cli

import (
	"context"

	"github.com/zadykian/ws/engine"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
)

// StateRead is the state screen's read of ws, by the remote script's report, which open reaches ws
// for. The session stays once found, as a new one would hold sudo's ticket until the screen ends.
// Each read asks ws's status again, as a run may have changed it.
func StateRead(open SessionOpen, version string) tui.StateReader {
	var ws *session.Session
	return func(ctx context.Context, b *tui.Bridge) ([]tui.StateSection, error) {
		if ws == nil {
			found, err := open(ctx, b)
			if err != nil {
				return nil, err
			}
			ws = found
		}
		states, err := ws.Restate(ctx, version)
		return engine.RoleSections(states), err
	}
}
