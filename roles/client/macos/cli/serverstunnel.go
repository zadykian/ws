package cli

import (
	"context"
	"errors"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/client/macos/internal/launchd"
	"github.com/zadykian/ws/roles/client/macos/tunnel/plan"
)

// tunnelOf says in a word where a server's tunnel stands, reading as the user: launchd's system
// domain and the daemon's state file are both world-readable. A server with no plist has no daemon
// yet, and one that launchd does not have loaded is stopped. A running daemon shows its tier and
// endpoint, or starting before it has written one.
func tunnelOf(ctx context.Context, m effect.Machine, name string) string {
	d := launchd.For(name)
	if _, err := m.Stat(d.Plist); err != nil {
		return "absent"
	}
	if _, err := m.Run(ctx, "launchctl", "print", "system/"+d.Label); err != nil {
		return "stopped"
	}
	text, err := m.ReadFile(d.State)
	if errors.Is(err, fs.ErrNotExist) {
		return "starting"
	}
	if err != nil {
		return "unreadable"
	}
	s, err := plan.ParseState(string(text))
	if err != nil {
		return "unreadable"
	}
	if s.Endpoint == "" {
		return string(s.Tier)
	}
	return string(s.Tier) + " " + s.Endpoint
}
