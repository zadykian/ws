package wizard

import (
	"cmp"
	"context"
	"errors"
	"strings"

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
	// user returns WS_USER, whose claude and git config the stages read.
	user() (string, error)
	// claude runs claude auth login as WS_USER at the terminal.
	claude(ctx context.Context) error
	// rerun says how to run ws server apply again, for a stop that needs it.
	rerun() string
	// hand adds what stays by hand of this way's ws, as the status s read it.
	hand(h *Hand, s wsStatus)
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

// rerun says how to run ws server apply again, for a stop that needs it.
func (s *Session) rerun() string { return s.reach.rerun() }

// user returns WS_USER, whose claude and git config the stages read.
func (s *Session) user() (string, error) { return s.reach.user() }

// claude runs claude's login as WS_USER. The login changes what the status reads, outside the
// scripts.
func (s *Session) claude(ctx context.Context) error {
	s.remote.stale()
	return s.reach.claude(ctx)
}

// State returns the roles of ws's report, as Remote's State does.
func (s *Session) State(ctx context.Context, version string) ([]apply.RoleState, error) {
	return s.remote.State(ctx, version)
}

// Hand adds what stays by hand of ws to the list, from the remote script's status.
func (s *Session) Hand(ctx context.Context, h *Hand) {
	st, _ := s.status(ctx)
	s.reach.hand(h, st)
}

// Files returns the files script's Remote, which reads and writes host_vars.
func (s *Session) Files() *Remote { return s.files }

// Secrets runs ws server secrets ARGS on ws as root, with in as its stdin, and returns its stdout. A
// value goes in in alone: ps shows the arguments. No script has an op for it, as the command is
// ws's own, and the status stays stale after it, as it may change what the status reads. A failed
// ssh says that ws is out of reach; else the error is what ws printed.
func (s *Session) Secrets(ctx context.Context, in []byte, args ...string) ([]byte, error) {
	s.remote.stale()
	out, err := s.remote.Shell.Run(ctx, in, append([]string{wsPath, "server", "secrets"}, args...)...)
	switch {
	case err == nil:
		return out, nil
	case strings.Contains(err.Error(), "exit status 255"):
		return nil, errors.New("ws is out of reach: ssh to ws failed: " + stderrOf(err))
	}
	return nil, errors.New(cmp.Or(stderrOf(err), err.Error()))
}

// Restate is State with ws's status read again, for a screen that reads ws more than once: a run
// may have changed ws since.
func (s *Session) Restate(ctx context.Context, version string) ([]apply.RoleState, error) {
	s.remote.stale()
	return s.State(ctx, version)
}
