package wizard

import (
	"cmp"
	"context"
	"errors"
	"strings"
)

// wsPath is ws's binary on ws, where the Install stage puts it.
const wsPath = "/usr/local/bin/ws"

// Secrets runs ws server secrets ARGS on ws as root, with in as its stdin, and returns its stdout. A
// value goes in in alone: ssh joins args into one line for ws's shell, so each goes quoted, and ps
// shows the line. A failed ssh says that ws is out of reach; else the error is what ws printed.
func (r *Remote) Secrets(ctx context.Context, in []byte, args ...string) ([]byte, error) {
	r.read = false
	cmd := []string{wsPath, "server", "secrets"}
	for _, a := range args {
		cmd = append(cmd, "'"+strings.ReplaceAll(a, "'", `'\''`)+"'")
	}
	argv, stdin, err := r.command(cmd, in)
	if err != nil {
		return nil, err
	}
	out, err := r.Shell.SSH.RunInput(ctx, stdin, argv...)
	switch {
	case err == nil:
		return out, nil
	case strings.Contains(err.Error(), "exit status 255"):
		return nil, errors.New("ws is out of reach: ssh to ws failed: " + stderrOf(err))
	}
	return nil, errors.New(cmp.Or(stderrOf(err), err.Error()))
}
