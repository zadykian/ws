package wizard

import (
	"context"

	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/internal/server/apply"
	"github.com/zadykian/ws/step"
)

// Session is the one way to ws of a wizard's run: the Mac's over ssh, or ws's own on the machine
// itself. It runs the remote script and the files script, which share ws's status. The stages take
// the session, never another stage, for what they read of ws.
type Session struct {
	reach  Reach
	remote *Remote
	files  *Remote
}

// Reach is how a Session reaches ws: MacReach or WsReach.
type Reach interface {
	// shell runs a command as root on ws.
	shell() Shell
	// want reads the target and WS_USER that ws's run is to leave: mac.conf's on the Mac, and
	// host_vars' on ws.
	want() (conf.Conf, error)
}

// Scripts are what a Session runs on ws: the remote script and the files script. Events hears a
// run's events, as Remote's does.
type Scripts struct {
	Remote, Files []byte
	Events        step.Observer
}

// NewSession returns the session of r, which runs sc's scripts.
func NewSession(r Reach, sc Scripts) *Session {
	remote := &Remote{Shell: r.shell(), Script: sc.Remote, Events: sc.Events}
	return &Session{reach: r, remote: remote, files: remote.Beside(sc.Files)}
}

// status reads ws's status once, until a change through either script.
func (s *Session) status(ctx context.Context) (wsStatus, bool) { return s.remote.status(ctx) }

// known returns what status last read, with no read of its own.
func (s *Session) known() wsStatus { return s.remote.known() }

// want reads what ws's run is to leave.
func (s *Session) want() (conf.Conf, error) { return s.reach.want() }

// State returns the roles of ws's report, as Remote's State does.
func (s *Session) State(ctx context.Context, version string) ([]apply.RoleState, error) {
	return s.remote.State(ctx, version)
}
