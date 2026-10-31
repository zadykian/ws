package setup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/mac/internal/step"
)

// publicKey asks root's copy of ws for the Mac's public key. It is "" where there is no copy yet,
// as a check before the first run, or where the key is to be made. It is "" too where the copy, or
// a directory above it, is to change, as the sh fixed or replaced them before it ran the copy.
func (t *Tunnel) publicKey(ctx context.Context) (string, error) {
	if t.keyMade || t.untrusted {
		return "", nil
	}
	if _, err := t.M.RootAttrs(tunnelLibexec + "/ws"); errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	out, err := t.M.RunRoot(ctx, tunnelLibexec+"/ws", "tunnel", "key")
	if err != nil {
		return "", fmt.Errorf("cannot read the Mac's public key: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// pubFile is the Mac's public key where the wizard's Tunnel stage reads it without sudo.
func (t *Tunnel) pubFile() string { return t.Home + "/.config/ws/tunnel.pub" }

// tell plans ~/.config/ws/tunnel.pub, once the key is made, and keeps the key for Finish.
func (t *Tunnel) tell(ctx context.Context) ([]step.Change, error) {
	public, err := t.publicKey(ctx)
	if err != nil || public == "" {
		return nil, err
	}
	t.public = public
	var cs []step.Change
	for _, d := range []struct {
		path string
		mode fs.FileMode
	}{{t.Home + "/.config", 0o755}, {t.Home + "/.config/ws", 0o700}} {
		if info, err := t.M.Stat(d.path); errors.Is(err, fs.ErrNotExist) {
			cs = append(cs, MakeDir{d.path, d.mode})
		} else if err != nil {
			return nil, err
		} else if !info.Mode.IsDir() {
			return nil, fmt.Errorf("%s is not a directory", d.path)
		}
	}
	want := []byte(public + "\n")
	have, err := t.M.ReadFile(t.pubFile())
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return append(cs, Write{Path: t.pubFile(), After: want, Mode: 0o644}), nil
	case err != nil:
		return nil, err
	case !bytes.Equal(have, want):
		return append(cs, Write{Path: t.pubFile(), Before: have, After: want, Exists: true, Mode: 0o644}), nil
	}
	return cs, nil
}

// publish writes tunnel.pub after the changes, where the key or root's copy was missing when the
// step planned, so that the plan after it is empty.
func (t *Tunnel) publish(ctx context.Context) error {
	t.keyMade = false
	t.untrusted = false
	cs, err := t.tell(ctx)
	if err != nil {
		return err
	}
	for _, c := range cs {
		if err := Apply(ctx, t.M, t.Out, c); err != nil {
			return err
		}
	}
	return nil
}
