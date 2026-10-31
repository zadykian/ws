package wizard

import "context"

// Remote runs the remote script's operations as root on ws, through ssh, which reaches ws at the
// address that the Host key stage asked, where it asked one.
type Remote struct {
	SSH SSH
	// Script is wizard-remote.sh, which sh -s reads from standard input.
	Script []byte
}

// args returns ssh's arguments for the operation. No argument holds a blank or a quote.
func (*Remote) args(op []string) []string {
	return append([]string{"-o", "BatchMode=yes", "ws-ssh", "sh", "-s", "--"}, op...)
}

// Do runs the operation and returns its output.
func (r *Remote) Do(ctx context.Context, op ...string) ([]byte, error) {
	return r.SSH.RunInput(ctx, r.Script, r.args(op)...)
}

// Pass is Do for an operation that goes on, such as run, whose output goes to Stderr as it comes.
func (r *Remote) Pass(ctx context.Context, op ...string) error {
	return r.SSH.PassInput(ctx, r.Script, r.args(op)...)
}
