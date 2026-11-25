package tunnel

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/internal/wgkey"
)

// peersDir holds the public key of each peer of wg0, as NAME.pub.
const peersDir = "/etc/wireguard"

// keyFile is the server's private key, which networkd reads for wg0.
const keyFile = "/etc/systemd/network/60-wg0.key"

// keyMode is the key's mode: networkd reads it as the user systemd-network, so the file is
// root:systemd-network 0640, as systemd.netdev(5) asks. A credential in /etc/credstore, which
// could be 0600, would be read only as networkd starts, so the run that made it would restart
// networkd under the uplink.
const keyMode fs.FileMode = 0o640

// Key is the server's private key, which the run makes on the machine where none exists and never
// shows or diffs. A key put back by hand may lack the owner, group and mode. networkd cannot read a
// root:root 0600 key as it next creates wg0, and anyone can read one written under umask 022.
type Key struct {
	M effect.Machine
	// Out takes a new key's public key, which the Mac's end needs. mise shows a unit's lines only
	// where it changes, so the run that makes the key says it.
	Out effect.Out
	// Random is the source of a new key's bytes, crypto/rand's where nil.
	Random io.Reader
	// File is the key's path, keyFile where empty: a guest's tunnel on the host has a key of its own.
	File string
}

// path is the key's file.
func (k Key) path() string {
	if k.File == "" {
		return keyFile
	}
	return k.File
}

// key is Key with the id of the group systemd-network, read as it plans.
type key struct {
	Key
	gid int
}

type makeKey struct{ key }

func (makeKey) String() string { return "make the server's private key (content not shown)" }

func (c makeKey) Apply(context.Context) error {
	random := c.Random
	if random == nil {
		random = rand.Reader
	}
	private, err := wgkey.Generate(random)
	if err != nil {
		return err
	}
	public, err := wgkey.Public(private)
	if err != nil {
		return err
	}
	if err := c.M.WriteAtomic(c.path(), []byte(private+"\n"), effect.Attrs{GID: c.gid, Mode: keyMode}); err != nil {
		return err
	}
	c.Out.Say("ws's WireGuard public key, for WS_PUBLIC_KEY in a server's values.yml: %s", public)
	return nil
}

type setKeyMode struct{ key }

func (c setKeyMode) String() string {
	return fmt.Sprintf("set the owner root:%s and the mode %o of %s", networkGroup, keyMode, c.path())
}

func (c setKeyMode) Apply(context.Context) error {
	if err := c.M.Chown(c.path(), 0, c.gid); err != nil {
		return err
	}
	return c.M.Chmod(c.path(), keyMode)
}

// Plan plans the key where none exists, or its owner, group and mode where they differ. A file that
// holds no private key is an error, as is a machine without the group systemd-network.
func (k Key) Plan(context.Context) ([]resource.Change, error) {
	gid, err := groupID(k.M)
	if err != nil {
		return nil, err
	}
	kg := key{k, gid}
	text, err := k.M.ReadRootFile(k.path())
	if errors.Is(err, fs.ErrNotExist) {
		return []resource.Change{makeKey{kg}}, nil
	}
	if err != nil {
		return nil, err
	}
	if _, err := wgkey.Public(strings.TrimSpace(string(text))); err != nil {
		return nil, fmt.Errorf("%s holds no WireGuard private key: %w", k.path(), err)
	}
	got, err := k.M.RootAttrs(k.path())
	if err != nil {
		return nil, err
	}
	if got != (effect.Attrs{GID: gid, Mode: keyMode}) {
		return []resource.Change{setKeyMode{kg}}, nil
	}
	return nil, nil
}

// publicKeyOf returns the public key of the private key in file, and whether the file exists.
func publicKeyOf(m effect.Machine, file string) (string, bool, error) {
	text, err := m.ReadRootFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	public, err := wgkey.Public(strings.TrimSpace(string(text)))
	return public, err == nil, err
}
