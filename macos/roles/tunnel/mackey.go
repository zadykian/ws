package tunnel

import (
	"context"
	"io"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/internal/etcws"
	"github.com/zadykian/ws/macos/roles/tunnel/daemon/settings"
)

// tunnelKeyFile is the Mac's WireGuard key, in a file only root reads. pubFile is its public half,
// root's and readable by all, so that a read takes no sudo.
const (
	tunnelKeyFile = Dir + "/private.key"
	pubFile       = etcws.Dir + "/tunnel.pub"
)

// MakeKey makes the Mac's WireGuard key where none is, as root, and returns the public key. It
// makes /etc/ws and /etc/ws/tunnel first, readable by all, so that the unit's check finds the key
// with no root. The key stays root's at 0600, and /etc/ws/tunnel.pub, readable by all, holds the
// public key for the commands that make the Mac a server's peer.
func MakeKey(ctx context.Context, m effect.Machine, random io.Reader) (string, error) {
	if err := settle(ctx, etcws.Top(m), file.Dir{M: m, Path: Dir, Mode: etcws.DirMode}); err != nil {
		return "", err
	}
	public, err := settings.Key(ctx, m, random, tunnelKeyFile)
	if err != nil {
		return "", err
	}
	// A Dir sets a file's owner and mode as a directory's, and never makes it here, as it exists.
	return public, settle(ctx, file.Dir{M: m, Path: tunnelKeyFile, Mode: 0o600},
		file.Public{M: m, Path: pubFile, Content: []byte(public + "\n"), Mode: etcws.FileMode})
}

// settle plans and applies each resource in order, so that a directory exists before what it holds
// plans.
func settle(ctx context.Context, rs ...resource.Resource) error {
	for _, r := range rs {
		cs, err := resource.Append(ctx, nil, r)
		if err != nil {
			return err
		}
		if err := (resource.Group{}).Apply(ctx, step.Plan{Changes: cs}); err != nil {
			return err
		}
	}
	return nil
}
