package wizard

import (
	"context"
	"strings"

	"github.com/zadykian/ws/internal/server/apply"
	"github.com/zadykian/ws/step"
)

// Remote runs the remote script's operations as root on ws, through ssh, which reaches ws at the
// address that the Host key stage asked, where it asked one.
type Remote struct {
	SSH SSH
	// Script is wizard-remote.sh, which sh -s reads from standard input.
	Script []byte
	// Sudo is how a script reaches root. Nil runs it as the login.
	Sudo *Sudo
	// Events hears a run's events. Nil drops them. It hears them on a goroutine of ssh's, while the
	// stage waits, and hears them again where a follow replays the log.
	Events step.Observer

	read   bool
	status map[string]string
}

// args returns ssh's arguments and stdin for the operation. No argument of op holds a blank or a quote.
func (r *Remote) args(op []string) ([]string, []byte, error) {
	return r.command(append([]string{"sh", "-s", "--"}, op...), r.Script)
}

// command returns ssh's arguments and stdin for cmd as root, with in as its stdin.
func (r *Remote) command(cmd []string, in []byte) ([]string, []byte, error) {
	cmd, in, err := r.Sudo.rooted(cmd, in)
	return append([]string{"-o", "BatchMode=yes", "ws"}, cmd...), in, err
}

// Do runs the operation, and has the next Status read ws again, as the operation may change it.
func (r *Remote) Do(ctx context.Context, op ...string) ([]byte, error) {
	r.read = false
	return r.query(ctx, op...)
}

// query runs an operation that changes nothing, so what Status read stands.
func (r *Remote) query(ctx context.Context, op ...string) ([]byte, error) {
	args, in, err := r.args(op)
	if err != nil {
		return nil, err
	}
	return r.SSH.RunInput(ctx, in, args...)
}

// Pass is Do for an operation that goes on, such as install, whose output goes to Stderr as it comes.
func (r *Remote) Pass(ctx context.Context, op ...string) error {
	r.read = false
	args, in, err := r.args(op)
	if err != nil {
		return err
	}
	return r.SSH.PassInput(ctx, in, args...)
}

// Follow is Pass for the operation run, with ws server apply's args. The events of the run's log go
// to Events, and its other lines to Stderr.
func (r *Remote) Follow(ctx context.Context, args ...string) error {
	r.read = false
	a, in, err := r.args(append([]string{"run"}, args...))
	if err != nil {
		return err
	}
	return r.SSH.PassLines(ctx, in, r.event, a...)
}

// event tells whether a line of a run's log is an event, and hands it on.
func (r *Remote) event(line string) bool {
	e, ok := apply.ParseEvent(line)
	if ok && r.Events != nil {
		r.Events.Observe(e)
	}
	return ok
}

// Put writes data to path on ws as root, mode 0600, through its stdin. path holds no blank or quote:
// the remote script named it.
func (r *Remote) Put(ctx context.Context, path string, data []byte) error {
	args, in, err := r.command([]string{"sh", "-c", "'umask 077 && cat >" + path + "'"}, data)
	if err != nil {
		return err
	}
	_, err = r.SSH.RunInput(ctx, in, args...)
	return err
}

// Status reads what the remote script's status prints once, until Do, Pass or Follow runs, and reports
// whether it read any: ssh or the script fails where root does not log in. A line is NAME VALUE.
func (r *Remote) Status(ctx context.Context) bool {
	if !r.read {
		out, err := r.Do(ctx, "status")
		r.read, r.status = true, nil
		if err == nil {
			r.status = map[string]string{}
			for _, l := range strings.Split(string(out), "\n") {
				if name, value, ok := strings.Cut(l, " "); ok {
					if _, dup := r.status[name]; !dup {
						r.status[name] = value
					}
				}
			}
		}
	}
	return len(r.status) > 0
}

// Get returns the value of NAME in what Status read, or an empty string.
func (r *Remote) Get(name string) string { return r.status[name] }

// Forget has the next Status read ws again, as after a session that changed it.
func (r *Remote) Forget() { r.read = false }
