package wizard

import (
	"context"

	"github.com/zadykian/ws/engine/core/effect"
)

// LocalShell runs a command on ws itself, as the user that runs ws, root for ws server ubuntu
// wizard. Its words reach the command as they are, with no shell between.
type LocalShell struct {
	M effect.Machine
	// User returns WS_USER, whose claude and git config the remote script reads as SUDO_USER's, as
	// sudo names the user whom the Mac's ssh logs in as. Nil, or an empty name, leaves SUDO_USER
	// unchanged: the user that ran sudo.
	User func() string
}

// argv returns cmd, past env where User names WS_USER.
func (l LocalShell) argv(cmd []string) []string {
	if l.User != nil {
		if u := l.User(); u != "" {
			return append([]string{"env", "SUDO_USER=" + u}, cmd...)
		}
	}
	return cmd
}

// Run runs cmd, and returns its standard output.
func (l LocalShell) Run(ctx context.Context, in []byte, cmd ...string) ([]byte, error) {
	cmd = l.argv(cmd)
	return l.M.RunInput(ctx, in, cmd[0], cmd[1:]...)
}

// Pass is Run with the output passed to Stderr as it comes.
func (l LocalShell) Pass(ctx context.Context, in []byte, cmd ...string) error {
	cmd = l.argv(cmd)
	return l.M.PassInput(ctx, in, cmd[0], cmd[1:]...)
}

// Lines is Pass with each line offered to take first. A Machine without effect.Lines passes every
// line on, as SSH's PassLines does.
func (l LocalShell) Lines(ctx context.Context, in []byte, take func(line string) bool, cmd ...string) error {
	if m, ok := l.M.(effect.Lines); ok {
		cmd = l.argv(cmd)
		return m.PassLines(ctx, in, take, cmd[0], cmd[1:]...)
	}
	return l.Pass(ctx, in, cmd...)
}
