// Package session is how the wizard reaches ws: one Session, over the Mac's ssh or on the machine
// itself, that runs the scripts which read and change ws. The scripts stay sh, as they run on ws
// before the ws binary is there.
package session

import (
	"cmp"
	"context"
	"errors"
	"github.com/zadykian/ws/engine/core/remote"
	guided "github.com/zadykian/ws/engine/core/wizard"
	"net/netip"
	"strings"

	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/engine/core/step"
)

// Ws is the side of ws, and the title of its list, in the wizard's locks and its list of what stays
// by hand.
const Ws guided.Side = "Server (Ubuntu)"

// SideOf returns the side of the server that the Mac names. Where it has no name, as in ws's own
// wizard, that is Ws. Else the list's title holds the name, as the Mac works on several servers.
func SideOf(server string) guided.Side {
	if server == "" {
		return Ws
	}
	return guided.Side("Server " + server + " (Ubuntu)")
}

// Session is the one way to ws of a wizard's run: the Mac's over ssh, or ws's own on the machine
// itself. It runs the remote script and the files script, which share ws's status. The stages take
// the session, never another stage, for what they read of ws.
type Session struct {
	reach Reach
	shell remote.Shell
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
	last      Status
}

// Reach is how a Session reaches ws: MacReach or WsReach.
type Reach interface {
	// Shell runs a command as root on ws.
	Shell() remote.Shell
	// Want reads the target, WS_USER, WS_SLOT and WS_PORT that ws's run is to leave: server.conf's
	// on the Mac, and host_vars' on ws.
	Want() (Want, error)
	// User returns WS_USER, whose claude and git config the stages read.
	User() (string, error)
	// Claude runs claude auth login as WS_USER at the terminal.
	Claude(ctx context.Context) error
	// Rerun says how to run ws server ubuntu apply again, for a stop that needs it.
	Rerun() string
	// Hand adds what stays by hand of this way's ws, as the status s read it.
	Hand(h *guided.Hand, s Status)
}

// Want is what ws's run is to leave: the target, lan or cloud, WS_USER, and wg0's address and port.
// The server reads no other key of server.conf. ws's own wizard reads no conf and leaves Address
// and Port zero, so its run writes no tunnel pair.
type Want struct {
	Target, User string
	// Address is wg0's address at the server's slot (WS_SLOT), and Port is WS_PORT as the conf
	// holds it, empty where it names none.
	Address netip.Prefix
	Port    string
}

// Scripts are what a Session runs on ws: the remote script and the files script. Events hears a
// run's events.
type Scripts struct {
	Remote, Files []byte
	Events        step.Observer
}

// New returns the session of r, which runs sc's scripts.
func New(r Reach, sc Scripts) *Session {
	return &Session{reach: r, shell: r.Shell(), remote: sc.Remote, files: sc.Files, events: sc.Events}
}

// Status reads what the remote script's status prints once, until an operation that changes ws, or
// claude's login, runs. It reports whether it read any: the Shell or the script fails where root is
// out of reach.
func (s *Session) Status(ctx context.Context) (Status, bool) {
	h := &s.held
	if !h.read {
		out, err := s.do(ctx, opStatus)
		h.read, h.got, h.last = true, err == nil && strings.TrimSpace(string(out)) != "", Status{}
		if h.got {
			h.last = parseStatus(string(out))
		}
	}
	return h.last, h.got
}

// Known returns what status last read, with no read of its own: zero before the first.
func (s *Session) Known() Status { return s.held.last }

// stale has the next status read ws again. An operation that changes ws marks it, and so does
// claude's login, which runs outside the scripts.
func (s *Session) stale() { s.held.read = false }

// Want reads what ws's run is to leave.
func (s *Session) Want() (Want, error) { return s.reach.Want() }

// Rerun says how to run ws server ubuntu apply again, for a stop that needs it.
func (s *Session) Rerun() string { return s.reach.Rerun() }

// User returns WS_USER, whose claude and git config the stages read.
func (s *Session) User() (string, error) { return s.reach.User() }

// Claude runs claude's login as WS_USER. The login changes what the status reads, outside the
// scripts.
func (s *Session) Claude(ctx context.Context) error {
	s.stale()
	return s.reach.Claude(ctx)
}

// Hand adds what stays by hand of ws to the list, from the remote script's status.
func (s *Session) Hand(ctx context.Context, h *guided.Hand) {
	st, _ := s.Status(ctx)
	s.reach.Hand(h, st)
}

// Secrets runs ws server ubuntu secrets ARGS on ws as root, with in as its stdin, and returns its
// stdout. A value goes in in alone: ps shows the arguments. No script has an op for it, as the
// command is ws's own, and the status stays stale after it, as it may change what the status reads.
// A failed ssh says that ws is out of reach; else the error is what ws printed.
func (s *Session) Secrets(ctx context.Context, in []byte, args ...string) ([]byte, error) {
	s.stale()
	out, err := s.shell.Run(ctx, in, append([]string{WsPath, "server", "ubuntu", "secrets"}, args...)...)
	switch {
	case err == nil:
		return out, nil
	case strings.Contains(err.Error(), "exit status 255"):
		return nil, errors.New("ws is out of reach: ssh to ws failed: " + remote.Stderr(err))
	}
	return nil, errors.New(cmp.Or(remote.Stderr(err), err.Error()))
}

// Restate is State with ws's status read again, for a screen that reads ws more than once: a run
// may have changed ws since.
func (s *Session) Restate(ctx context.Context, version string) ([]role.RoleState, error) {
	s.stale()
	return s.State(ctx, version)
}
