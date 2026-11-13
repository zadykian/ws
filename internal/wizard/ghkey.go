package wizard

import (
	"context"
	"strings"

	"github.com/zadykian/ws/roles/client/macos/setup"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
)

// noKey is the failure to read the Mac's key: its .pub is not there yet, or cannot be read.
type noKey struct{ err error }

func (e noKey) Error() string { return e.err.Error() }

func (e noKey) Unwrap() error { return e.err }

// key returns the .pub of the Mac's key for ws, as setup.ReadKey finds it from mac.conf, and its
// type and base64.
func (g GhCLI) key(ctx context.Context) (pub, key string, err error) {
	_, c, err := loadConf(confFile{g.Conf, stage.GitHub, g.M, g.Out})
	if err != nil {
		return "", "", err
	}
	k, err := setup.ReadKey(ctx, g.M, c, g.Home, setup.LoginKey)
	if err != nil {
		return "", "", noKey{err}
	}
	return k.File, k.Key.String(), nil
}

func (g *Gh) hostname(ctx context.Context) string {
	out, err := g.M.Run(ctx, "hostname", "-s")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
