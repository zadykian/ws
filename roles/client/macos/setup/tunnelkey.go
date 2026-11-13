package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/change"
)

// tunnelKeyFile is the Mac's WireGuard key, in a file only root reads.
const tunnelKeyFile = tunnelDir + "/private.key"

// KeyMake has root's ws make the Mac's WireGuard key, which is never shown to the user.
type KeyMake struct{}

func (KeyMake) String() string { return "would make the Mac's WireGuard key, " + tunnelKeyFile }

// key plans /etc/ws-tunnel, routes.d and the Mac's key: made where missing, and its owner and mode
// set where they differ. It sets keyMade.
func (t *Tunnel) key(ctx context.Context) ([]step.Change, error) {
	var cs []step.Change
	var err error
	for _, d := range []host.Dir{{M: t.M, Path: tunnelDir, Mode: 0o700}, {M: t.M, Path: tunnelDir + "/routes.d", Mode: 0o755}} {
		if cs, err = change.Plan(ctx, cs, d); err != nil {
			return nil, err
		}
	}
	_, err = t.M.RootAttrs(tunnelKeyFile)
	t.keyMade = errors.Is(err, fs.ErrNotExist)
	switch {
	case t.keyMade:
		return append(cs, KeyMake{}), nil
	case err != nil:
		return nil, err
	}
	// A Dir sets a file's owner and mode as a directory's, and never makes it here, as it exists.
	return change.Plan(ctx, cs, host.Dir{M: t.M, Path: tunnelKeyFile, Mode: 0o600})
}

func (t *Tunnel) makeKey(ctx context.Context) error {
	if _, err := t.M.RunRoot(ctx, tunnelLibexec+"/ws", "client", "macos", "tunnel", "key"); err != nil {
		return fmt.Errorf("cannot make the Mac's WireGuard key: %w", err)
	}
	t.madeKey = true
	t.Out.Say("made the Mac's WireGuard key, %s", tunnelKeyFile)
	return nil
}
