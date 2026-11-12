package network

import (
	"context"
	"errors"
	"os/exec"

	"github.com/zadykian/ws/engine/core/effect"
)

// runRoot runs a command through sudo and returns what it left. A command that exits with a code
// is a result, not an error: netplan's refusals are the check's to word.
func runRoot(ctx context.Context, m effect.Machine, name string, args ...string) Result {
	out, err := m.RunRoot(ctx, name, args...)
	var exit *exec.ExitError
	switch {
	case err == nil:
		return Result{Stdout: string(out)}
	case errors.As(err, &exit):
		return Result{RC: exit.ExitCode(), Stdout: string(out), Stderr: string(exit.Stderr)}
	default:
		// The error holds no standard error: a command that died on a signal has an ExitError.
		return Result{RC: -1, Msg: err.Error()}
	}
}
