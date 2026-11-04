package wizard

import (
	"context"
	"strings"
)

// Remote runs the remote script's operations as root on ws, through ssh, which reaches ws at the
// address that the Host key stage asked, where it asked one.
type Remote struct {
	SSH SSH
	// Script is wizard-remote.sh, which sh -s reads from standard input.
	Script []byte

	read   bool
	status map[string]string
}

// args returns ssh's arguments for the operation. No argument holds a blank or a quote.
func (*Remote) args(op []string) []string {
	return append([]string{"-o", "BatchMode=yes", "ws", "sh", "-s", "--"}, op...)
}

// Do runs the operation, and has the next Status read ws again, as the operation may change it.
func (r *Remote) Do(ctx context.Context, op ...string) ([]byte, error) {
	r.read = false
	return r.SSH.RunInput(ctx, r.Script, r.args(op)...)
}

// Pass is Do for an operation that goes on, such as run, whose output goes to Stderr as it comes.
func (r *Remote) Pass(ctx context.Context, op ...string) error {
	r.read = false
	return r.SSH.PassInput(ctx, r.Script, r.args(op)...)
}

// Status reads what the remote script's status prints once, until Do or Pass runs, and reports
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
