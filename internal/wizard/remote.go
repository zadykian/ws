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

	// held is what status read, which the Remotes that Beside returns share.
	held *heldStatus
}

// heldStatus is what status read of ws. read is set until a change through a Remote that shares
// it, and got where status read any.
type heldStatus struct {
	read, got bool
	last      wsStatus
}

// hold returns what r's status holds, made on the first call.
func (r *Remote) hold() *heldStatus {
	if r.held == nil {
		r.held = &heldStatus{}
	}
	return r.held
}

// Beside returns a Remote of the script over r's Shell, such as the files script's. A Do, Pass,
// Follow or Secrets through either leaves the status stale for both, as the files script changes
// what the status reads.
func (r *Remote) Beside(script []byte) *Remote {
	return &Remote{Shell: r.Shell, Script: script, held: r.hold()}
}

// script returns the command that runs the operation, with the script on its stdin.
func script(op []string) []string {
	return append([]string{"sh", "-s", "--"}, op...)
}

// Do runs the operation, and has the next status read ws again, as the operation may change it.
func (r *Remote) Do(ctx context.Context, op ...string) ([]byte, error) {
	r.Forget()
	return r.query(ctx, op...)
}

// query runs an operation that changes nothing, so what status read stands.
func (r *Remote) query(ctx context.Context, op ...string) ([]byte, error) {
	return r.Shell.Run(ctx, r.Script, script(op)...)
}

// Pass is Do for an operation that goes on, such as install, whose output goes to Stderr as it comes.
func (r *Remote) Pass(ctx context.Context, op ...string) error {
	r.Forget()
	return r.Shell.Pass(ctx, r.Script, script(op)...)
}

// Follow is Pass for the operation run, with ws server apply's args. The events of the run's log go
// to Events, and its other lines to Stderr.
func (r *Remote) Follow(ctx context.Context, args ...string) error {
	r.Forget()
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
	h := r.hold()
	if !h.read {
		out, err := r.query(ctx, "status")
		h.read, h.got, h.last = true, err == nil && strings.TrimSpace(string(out)) != "", wsStatus{}
		if h.got {
			h.last = parseStatus(string(out))
		}
	}
	return h.last, h.got
}

// known returns what status last read, with no read of its own: zero before the first.
func (r *Remote) known() wsStatus { return r.hold().last }

// Forget has the next status read ws again, as after a session that changed it.
func (r *Remote) Forget() { r.hold().read = false }
