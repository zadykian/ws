package wizard

import (
	"context"

	"github.com/zadykian/ws/effect"
)

// talk runs a command, never as root, that talks to the maintainer: by f, which gives it the terminal, or by m's
// Pass where f is nil.
func talk(ctx context.Context, f effect.Foreground, m effect.Machine, name string, args ...string) error {
	if f == nil {
		return m.Pass(ctx, false, name, args...)
	}
	return f.PassForeground(ctx, false, name, args...)
}

// RunInput runs ssh with the arguments and the bytes as standard input, and returns its standard
// output.
func (s SSH) RunInput(ctx context.Context, stdin []byte, args ...string) ([]byte, error) {
	return s.M.RunInput(ctx, stdin, "ssh", s.argv(ctx, args)...)
}

// PassInput is RunInput for a command that goes on, with its output passed to Stderr as it comes.
func (s SSH) PassInput(ctx context.Context, stdin []byte, args ...string) error {
	return s.M.PassInput(ctx, stdin, "ssh", s.argv(ctx, args)...)
}

// PassLines is PassInput with each line of the output offered to take first. A Machine without
// effect.Lines passes every line on: a run goes on, and its reader misses the lines.
func (s SSH) PassLines(ctx context.Context, stdin []byte, take func(line string) bool, args ...string) error {
	if m, ok := s.M.(effect.Lines); ok {
		return m.PassLines(ctx, stdin, take, "ssh", s.argv(ctx, args)...)
	}
	return s.PassInput(ctx, stdin, args...)
}

// Pass runs ssh with the arguments, with its output passed to Stderr and standard input kept, for a
// session at the terminal.
func (s SSH) Pass(ctx context.Context, args ...string) error {
	return talk(ctx, s.F, s.M, "ssh", s.argv(ctx, args)...)
}
