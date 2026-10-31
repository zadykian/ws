package setup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/zadykian/ws/mac/internal/conf"
	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
)

const (
	// tunnelLibexec holds root's copy of ws, which the daemon runs. The user owns ~/.local/bin, so a
	// root daemon that ran that file would let anything that runs as the user become root at its
	// next start.
	tunnelLibexec = "/usr/local/libexec/ws-tunnel"
)

// Tunnel sets up the tunnel to ws: ws in ~/.local/bin and root's copy of it, the Mac's WireGuard
// key, and with WS_PUBLIC_KEY, tunnel.conf and the LaunchDaemon that runs `ws tunnel run`. Without
// WS_PUBLIC_KEY, ws's key, it installs no daemon yet.
type Tunnel struct {
	Conf conf.Conf
	Home string
	// Plist is the path of mac/com.github.zadykian.ws-tunnel.plist.
	Plist string
	// Self returns the bytes of this ws, which a build from the clone made. Nil means the
	// executable's file.
	Self func() ([]byte, error)
	M    effect.Machine
	Out  effect.Out
	// Check is set under --check.
	Check bool

	copied bool // root's copy of ws is to change, which the daemon restarts for
	// untrusted is set while the directories above root's copy, or the copy, are planned to change:
	// the copy is not run as root until then.
	untrusted bool
}

// Name is tunnel.
func (*Tunnel) Name() string { return "tunnel" }

// WaitsOn is nothing.
func (*Tunnel) WaitsOn() string { return "" }

func (t *Tunnel) self() ([]byte, error) {
	if t.Self != nil {
		return t.Self()
	}
	path, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path) //nolint:gosec // the running program's own file
}

// Install puts the bytes of ws in the user's file, through a new file renamed over it, so that a
// ws that runs from there keeps running. It stands for the build that setup.sh did.
type Install struct {
	Path string
	Data []byte
}

func (i Install) String() string { return "would build ws into " + i.Path }

// program plans ~/.local/bin/ws, which the copy for root comes from.
func (t *Tunnel) program(self []byte) (step.Change, error) {
	path := t.Home + "/.local/bin/ws"
	have, err := t.M.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if err == nil && bytes.Equal(have, self) {
		return nil, nil
	}
	return Install{path, self}, nil
}

// Plan plans ws and root's copy.
func (t *Tunnel) Plan(_ context.Context) (step.Plan, error) {
	self, err := t.self()
	if err != nil {
		return step.Plan{}, fmt.Errorf("cannot read ws: %w", err)
	}
	cs, err := t.programAndCopy(self)
	if err != nil {
		return step.Plan{}, err
	}
	return step.Plan{Changes: cs}, nil
}
