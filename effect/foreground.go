package effect

import (
	"context"
	"os"
)

// PassForeground is Pass with ws's own standard input and error, past Stdin and Stderr, which a
// screen points away from the terminal. Its output stays off ws's standard output, as Pass's does.
func (m OSMachine) PassForeground(ctx context.Context, root bool, name string, args ...string) error {
	cmd := m.command(ctx, m.argv(root, name, args))
	cmd.Env = nil // offTerminal is for a command off the terminal.
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stderr, os.Stderr
	if err := cmd.Run(); err != nil {
		return commandError(cmd, err)
	}
	return nil
}
