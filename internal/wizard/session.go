package wizard

import (
	"cmp"
	"context"
	"errors"
	"strings"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/roles/server/ubuntu/apply"
)

// Session is the one way to ws of a wizard's run: the Mac's over ssh, or ws's own on the machine
// itself. It runs the remote script and the files script, which share ws's status. The stages take
// the session, never another stage, for what they read of ws.
type Session struct {
	reach Reach
	shell Shell
	// remote and files are the scripts, which sh -s reads from standard input.
	remote, files []byte
	// events hears a run's events. Nil drops them. It hears them on a goroutine of the Shell's,
	// while the stage waits, and hears them again where a follow replays the log.
	events step.Observer
	held   heldStatus
}

// heldStatus is what status read of ws. read is set until a change, and got where status read any.
type heldStatus struct {
	read, got bool
	last      wsStatus
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
// run's events.
type Scripts struct {
	Remote, Files []byte
	Events        step.Observer
}

// NewSession returns the session of r, which runs sc's scripts.
func NewSession(r Reach, sc Scripts) *Session {
	return &Session{reach: r, shell: r.shell(), remote: sc.Remote, files: sc.Files, events: sc.Events}
}

// status reads what the remote script's status prints once, until an operation that changes ws, or
// claude's login, runs. It reports whether it read any: the Shell or the script fails where root is
// out of reach.
func (s *Session) status(ctx context.Context) (wsStatus, bool) {
	h := &s.held
	if !h.read {
		out, err := s.do(ctx, opStatus)
		h.read, h.got, h.last = true, err == nil && strings.TrimSpace(string(out)) != "", wsStatus{}
		if h.got {
			h.last = parseStatus(string(out))
		}
	}
	return h.last, h.got
}

// known returns what status last read, with no read of its own: zero before the first.
func (s *Session) known() wsStatus { return s.held.last }

// stale has the next status read ws again. An operation that changes ws marks it, and so does
// claude's login, which runs outside the scripts.
func (s *Session) stale() { s.held.read = false }

// want reads what ws's run is to leave.
func (s *Session) want() (conf.Conf, error) { return s.reach.want() }

// rerun says how to run ws server apply again, for a stop that needs it.
func (s *Session) rerun() string { return s.reach.rerun() }

// user returns WS_USER, whose claude and git config the stages read.
func (s *Session) user() (string, error) { return s.reach.user() }

// claude runs claude's login as WS_USER. The login changes what the status reads, outside the
// scripts.
func (s *Session) claude(ctx context.Context) error {
	s.stale()
	return s.reach.claude(ctx)
}

// Hand adds what stays by hand of ws to the list, from the remote script's status.
func (s *Session) Hand(ctx context.Context, h *Hand) {
	st, _ := s.status(ctx)
	s.reach.hand(h, st)
}

// Secrets runs ws server secrets ARGS on ws as root, with in as its stdin, and returns its stdout. A
// value goes in in alone: ps shows the arguments. No script has an op for it, as the command is
// ws's own, and the status stays stale after it, as it may change what the status reads. A failed
// ssh says that ws is out of reach; else the error is what ws printed.
func (s *Session) Secrets(ctx context.Context, in []byte, args ...string) ([]byte, error) {
	s.stale()
	out, err := s.shell.Run(ctx, in, append([]string{wsPath, "server", "secrets"}, args...)...)
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
	s.stale()
	return s.State(ctx, version)
}
