package wizard

import "context"

// RunInput runs ssh with the arguments and the bytes as standard input, and returns its standard
// output.
func (s SSH) RunInput(ctx context.Context, stdin []byte, args ...string) ([]byte, error) {
	return s.M.RunInput(ctx, stdin, "ssh", s.argv(ctx, args)...)
}

// PassInput is RunInput for a command that goes on, with its output passed to Stderr as it comes.
func (s SSH) PassInput(ctx context.Context, stdin []byte, args ...string) error {
	return s.M.PassInput(ctx, stdin, "ssh", s.argv(ctx, args)...)
}

// Pass runs ssh with the arguments, with its output passed to Stderr and standard input kept, for a
// session at the terminal.
func (s SSH) Pass(ctx context.Context, args ...string) error {
	return s.M.Pass(ctx, false, "ssh", s.argv(ctx, args)...)
}
