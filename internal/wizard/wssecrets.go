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
// value goes in in alone: ps shows the arguments. A failed ssh says that ws is out of reach; else the
// error is what ws printed.
func (r *Remote) Secrets(ctx context.Context, in []byte, args ...string) ([]byte, error) {
	r.Forget()
	out, err := r.Shell.Run(ctx, in, append([]string{wsPath, "server", "secrets"}, args...)...)
	switch {
	case err == nil:
		return out, nil
	case strings.Contains(err.Error(), "exit status 255"):
		return nil, errors.New("ws is out of reach: ssh to ws failed: " + stderrOf(err))
	}
	return nil, errors.New(cmp.Or(stderrOf(err), err.Error()))
}
