package tunnel

import (
	"context"
	"errors"
	"io/fs"

	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/change"
	"github.com/zadykian/ws/roles/client/macos/internal/launchd"
)

// tunnelKeyFile is the Mac's WireGuard key, in a file only root reads.
const tunnelKeyFile = launchd.Dir + "/private.key"

// KeyMake has root's ws make the Mac's WireGuard key, which is never shown to the user.
type KeyMake struct{ t *Tunnel }

func (KeyMake) String() string { return "would make the Mac's WireGuard key, " + tunnelKeyFile }

// Apply has root's copy of ws make the key, and says so. It sets madeKey, for Finish.
func (k KeyMake) Apply(ctx context.Context) error {
	t := k.t
	if _, err := t.M.RunRoot(ctx, tunnelLibexec+"/ws", "client", "macos", "tunnel", "key"); err != nil {
		return err
	}
	t.madeKey = true
	t.Out.Say("made the Mac's WireGuard key, %s", tunnelKeyFile)
	return nil
}

// key plans /etc/ws-tunnel, routes.d and the Mac's key: made where missing, and its owner and mode
// set where they differ. It sets keyMade.
func (t *Tunnel) key(ctx context.Context) ([]step.Change, error) {
	var cs []step.Change
	var err error
	for _, d := range []file.Dir{{M: t.M, Path: launchd.Dir, Mode: 0o700}, {M: t.M, Path: launchd.Dir + "/routes.d", Mode: 0o755}} {
		if cs, err = change.Plan(ctx, cs, d); err != nil {
			return nil, err
		}
	}
	_, err = t.M.RootAttrs(tunnelKeyFile)
	t.keyMade = errors.Is(err, fs.ErrNotExist)
	switch {
	case t.keyMade:
		return append(cs, KeyMake{t}), nil
	case err != nil:
		return nil, err
	}
	// A Dir sets a file's owner and mode as a directory's, and never makes it here, as it exists.
	return change.Plan(ctx, cs, file.Dir{M: t.M, Path: tunnelKeyFile, Mode: 0o600})
}
