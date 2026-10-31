package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/mac/internal/step"
)

// tunnelKeyFile is the Mac's WireGuard key, in a file only root reads.
const tunnelKeyFile = tunnelEtc + "/private.key"

// KeyMake has root's ws make the Mac's WireGuard key, which is never shown to the user.
type KeyMake struct{}

func (KeyMake) String() string { return "would make the Mac's WireGuard key, " + tunnelKeyFile }

// key plans /etc/ws-tunnel, routes.d and the Mac's key: made where missing, and its owner and mode
// set where they differ. It sets keyMade.
func (t *Tunnel) key() ([]step.Change, error) {
	var cs []step.Change
	for _, d := range []struct {
		path string
		mode fs.FileMode
	}{{tunnelEtc, 0o700}, {tunnelEtc + "/routes.d", 0o755}} {
		c, err := rootDir(t.M, d.path, d.mode)
		if err != nil {
			return nil, err
		}
		if c != nil {
			cs = append(cs, c)
		}
	}
	t.keyMade = false
	a, err := t.M.RootAttrs(tunnelKeyFile)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		t.keyMade = true
		return append(cs, KeyMake{}), nil
	case err != nil:
		return nil, err
	case a.UID != 0 || a.GID != 0 || a.Mode.Perm() != 0o600:
		cs = append(cs, RootFix{tunnelKeyFile, 0o600, attrsText(a)})
	}
	return cs, nil
}

func (t *Tunnel) makeKey(ctx context.Context) error {
	if _, err := t.M.RunRoot(ctx, tunnelLibexec+"/ws", "tunnel", "key"); err != nil {
		return fmt.Errorf("cannot make the Mac's WireGuard key: %w", err)
	}
	t.Out.Say("made the Mac's WireGuard key, %s", tunnelKeyFile)
	return nil
}
