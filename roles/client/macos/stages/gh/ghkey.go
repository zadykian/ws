package gh

import (
	"context"
	"strings"

	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/internal/keys"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
)

// noKey is the failure to read the Mac's key: its .pub is not there yet, or cannot be read.
type noKey struct{ err error }

func (e noKey) Error() string { return e.err.Error() }

func (e noKey) Unwrap() error { return e.err }

// Key returns the .pub of the Mac's key for ws, as keys.ReadKey finds it from mac.conf, and its
// type and base64.
func (g CLI) Key(ctx context.Context) (pub, key string, err error) {
	_, c, err := (conf.File{Path: g.Conf, Stage: stage.GitHub, M: g.M, Out: g.Out}).Load()
	if err != nil {
		return "", "", err
	}
	k, err := keys.ReadKey(ctx, g.M, c, g.Home, keys.LoginKey)
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
