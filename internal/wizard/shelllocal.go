package wizard

import (
	"context"

	"github.com/zadykian/ws/effect"
)

// LocalShell runs a command on ws itself, as the user that runs ws, root for ws server wizard. Its
// words reach the command as they are, with no shell between.
type LocalShell struct{ M effect.Machine }

// Run runs cmd, and returns its standard output.
func (l LocalShell) Run(ctx context.Context, in []byte, cmd ...string) ([]byte, error) {
	return l.M.RunInput(ctx, in, cmd[0], cmd[1:]...)
}

// Pass is Run with the output passed to Stderr as it comes.
func (l LocalShell) Pass(ctx context.Context, in []byte, cmd ...string) error {
	return l.M.PassInput(ctx, in, cmd[0], cmd[1:]...)
}

// Lines is Pass with each line offered to take first. A Machine without effect.Lines passes every
// line on, as SSH's PassLines does.
func (l LocalShell) Lines(ctx context.Context, in []byte, take func(line string) bool, cmd ...string) error {
	if m, ok := l.M.(effect.Lines); ok {
		return m.PassLines(ctx, in, take, cmd[0], cmd[1:]...)
	}
	return l.Pass(ctx, in, cmd...)
}

// Put writes data to path, mode 0600.
func (l LocalShell) Put(_ context.Context, path string, data []byte) error {
	return l.M.WriteFile(path, data, 0o600)
}
