package cli

import (
	"context"
	"github.com/zadykian/ws/roles/client/macos/stages/host"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
	ubuntucli "github.com/zadykian/ws/roles/server/ubuntu/cli"
)

// macState is the state screen's read of side's machine, and what ends it. ws's reach takes sudo, the
// scan's, which the run takes after it, and lets the lock go with the screen.
func macState(p enginecli.Env, server *string, side string, sudo *host.Sudo) (tui.StateReader, func(context.Context)) {
	if side == string(stage.Mac) {
		return macRead(p, server), nil
	}
	reach := &varsReach{p: p, server: *server, sudo: sudo}
	return ubuntucli.StateRead(reach.reach, p.Version), reach.close
}
