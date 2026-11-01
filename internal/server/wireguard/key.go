package wireguard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/wgkey"
)

// keyMode is the key's mode: networkd reads it as the user systemd-network, so the file is
// root:systemd-network 0640, as systemd.netdev(5) asks. A credential in /etc/credstore, which
// could be 0600, would be read only as networkd starts, so the run that made it would restart
// networkd under the uplink.
const keyMode fs.FileMode = 0o640

// key is the server's private key, which the run makes on the machine where it is missing and
// never shows or diffs. A key put back by hand may lack the owner, group and mode: networkd cannot
// read a root:root 0600 key as it next creates wg0, and anyone can read one written under umask
// 022.
type key struct {
	m      effect.Machine
	random io.Reader
	gid    int
}

type makeKey struct{ key }

func (makeKey) String() string { return "make the server's private key (content not shown)" }

func (c makeKey) Apply(context.Context) error {
	private, err := wgkey.Generate(c.random)
	if err != nil {
		return err
	}
	return c.m.WriteAtomic(keyFile, []byte(private+"\n"), effect.Attrs{GID: c.gid, Mode: keyMode})
}

type setKeyMode struct{ key }

func (setKeyMode) String() string {
	return fmt.Sprintf("set the owner root:%s and the mode %o of the server's private key", networkGroup, keyMode)
}

func (c setKeyMode) Apply(context.Context) error {
	if err := c.m.Chown(keyFile, 0, c.gid); err != nil {
		return err
	}
	return c.m.Chmod(keyFile, keyMode)
}

// Plan plans the key where it is missing, or its owner, group and mode where they differ. A file
// that holds no private key is an error.
func (k key) Plan(context.Context) ([]host.Change, error) {
	text, err := k.m.ReadRootFile(keyFile)
	if errors.Is(err, fs.ErrNotExist) {
		return []host.Change{makeKey{k}}, nil
	}
	if err != nil {
		return nil, err
	}
	if _, err := wgkey.Public(strings.TrimSpace(string(text))); err != nil {
		return nil, fmt.Errorf("%s holds no WireGuard private key: %w", keyFile, err)
	}
	got, err := k.m.RootAttrs(keyFile)
	if err != nil {
		return nil, err
	}
	if got != (effect.Attrs{GID: k.gid, Mode: keyMode}) {
		return []host.Change{setKeyMode{k}}, nil
	}
	return nil, nil
}

// serverPublicKey returns the public key of the private key in the file, and whether the file
// exists.
func serverPublicKey(m effect.Machine) (string, bool, error) {
	text, err := m.ReadRootFile(keyFile)
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	public, err := wgkey.Public(strings.TrimSpace(string(text)))
	return public, err == nil, err
}
