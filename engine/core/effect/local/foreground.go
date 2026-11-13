package local

import (
	"context"
	"os"
	"syscall"
	"time"
)

// foregroundGrace bounds a command's end past its context, under the 5 seconds that a screen gives
// a run's cleanup.
const foregroundGrace = 2 * time.Second

// PassForeground is Pass with ws's own standard input and error, past Stdin and Stderr, which a
// screen points away from the terminal. Its output stays off ws's standard output, as Pass's does.
// The context's end sends SIGTERM before a kill. ssh -t and ssh-keygen change the terminal, and
// only they can set it back: a screen would take a state they left as the one to restore.
func (m OSMachine) PassForeground(ctx context.Context, root bool, name string, args ...string) error {
	if err := m.ticket(root); err != nil {
		return err
	}
	cmd := m.command(ctx, m.argv(root, name, args))
	cmd.Env = nil // offTerminal is for a command off the terminal.
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stderr, os.Stderr
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	cmd.WaitDelay = foregroundGrace
	if err := cmd.Run(); err != nil {
		return commandError(cmd, err)
	}
	return nil
}
