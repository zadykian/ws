package wizard

import (
	"context"
	"strings"

	"github.com/zadykian/ws/internal/server/apply"
	"github.com/zadykian/ws/step"
)

// Remote runs the remote script's operations as root on ws, through its Shell.
type Remote struct {
	Shell Shell
	// Script is wizard-remote.sh, which sh -s reads from standard input.
	Script []byte
	// Events hears a run's events. Nil drops them. It hears them on a goroutine of the Shell's, while
	// the stage waits, and hears them again where a follow replays the log.
	Events step.Observer

	// read is set once status read ws, until Do, Pass or Follow runs; got is set where it read any.
	read, got bool
	last      wsStatus
}

// script returns the command that runs the operation, with the script on its stdin.
func script(op []string) []string {
	return append([]string{"sh", "-s", "--"}, op...)
}

// Do runs the operation, and has the next status read ws again, as the operation may change it.
func (r *Remote) Do(ctx context.Context, op ...string) ([]byte, error) {
	r.read = false
	return r.query(ctx, op...)
}

// query runs an operation that changes nothing, so what status read stands.
func (r *Remote) query(ctx context.Context, op ...string) ([]byte, error) {
	return r.Shell.Run(ctx, r.Script, script(op)...)
}

// Pass is Do for an operation that goes on, such as install, whose output goes to Stderr as it comes.
func (r *Remote) Pass(ctx context.Context, op ...string) error {
	r.read = false
	return r.Shell.Pass(ctx, r.Script, script(op)...)
}

// Follow is Pass for the operation run, with ws server apply's args. The events of the run's log go
// to Events, and its other lines to Stderr.
func (r *Remote) Follow(ctx context.Context, args ...string) error {
	r.read = false
	return r.Shell.Lines(ctx, r.Script, r.event, script(append([]string{"run"}, args...))...)
}

// event tells whether a line of a run's log is an event, and hands it on.
func (r *Remote) event(line string) bool {
	e, ok := apply.ParseEvent(line)
	if ok && r.Events != nil {
		r.Events.Observe(e)
	}
	return ok
}

// Put writes data to path on ws as root, mode 0600. path holds no blank or quote: the remote script
// named it.
func (r *Remote) Put(ctx context.Context, path string, data []byte) error {
	return r.Shell.Put(ctx, path, data)
}

// status reads what the remote script's status prints once, until Do, Pass or Follow runs. It
// reports whether it read any: the Shell or the script fails where root is out of reach.
func (r *Remote) status(ctx context.Context) (wsStatus, bool) {
	if !r.read {
		out, err := r.query(ctx, "status")
		r.read, r.got, r.last = true, err == nil && strings.TrimSpace(string(out)) != "", wsStatus{}
		if r.got {
			r.last = parseStatus(string(out))
		}
	}
	return r.last, r.got
}

// known returns what status last read, with no read of its own: zero before the first.
func (r *Remote) known() wsStatus { return r.last }

// Forget has the next status read ws again, as after a session that changed it.
func (r *Remote) Forget() { r.read = false }
