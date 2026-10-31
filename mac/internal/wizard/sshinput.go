package wizard

import "context"

// RunInput runs ssh with the arguments and the bytes as standard input, and returns its standard
// output.
func (s SSH) RunInput(ctx context.Context, stdin []byte, args ...string) ([]byte, error) {
	return s.M.RunInput(ctx, stdin, "ssh", s.argv(args)...)
}

// PassInput is RunInput for a command that goes on, with its output passed to Stderr as it comes.
func (s SSH) PassInput(ctx context.Context, stdin []byte, args ...string) error {
	return s.M.PassInput(ctx, stdin, "ssh", s.argv(args)...)
}
