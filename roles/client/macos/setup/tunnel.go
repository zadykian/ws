package setup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/internal/conf"
)

const (
	// tunnelLibexec holds root's copy of ws, which the daemon runs. The user owns ~/.local/bin, so a
	// root daemon that ran that file would let anything that runs as the user become root at its
	// next start.
	tunnelLibexec = "/usr/local/libexec/ws-tunnel"
	// tunnelDir holds the Mac's key, tunnel.conf and routes.d, which only root can write.
	tunnelDir = "/etc/ws-tunnel"
)

// Tunnel sets up the tunnel to ws: ws in ~/.local/bin and root's copy of it, and the Mac's
// WireGuard key. With WS_PUBLIC_KEY, it also sets up tunnel.conf and the LaunchDaemon that runs `ws
// mac tunnel run`. Without WS_PUBLIC_KEY, ws's key, it installs no daemon yet.
type Tunnel struct {
	Conf conf.Conf
	Home string
	// Plist is the text of mac/com.github.zadykian.ws-tunnel.plist.
	Plist []byte
	// Self returns the bytes of this ws, which a build from the clone made. Nil means the
	// executable's file.
	Self func() ([]byte, error)
	M    effect.Machine
	Out  effect.Out
	// Check is set under --check.
	Check bool
	// Text takes the instructions that make the Mac's key ws's peer, as ws mac apply printed them.
	Text io.Writer
	// Sleep waits between launchctl's tries. Nil means time.Sleep.
	Sleep func(time.Duration)

	copied  bool // root's copy of ws is to change, which the daemon restarts for
	keyMade bool // the Mac's key is to be made, which the daemon restarts for
	madeKey bool // this run made the key
	public  string
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
// ws that runs from there keeps running. It stands for the build that ws mac apply did.
type Install struct {
	Path string
	Data []byte
}

func (i Install) String() string { return "would build ws into " + i.Path }

// localBin is ~/.local/bin, which holds ws's program. A run makes it where missing, and leaves its mode.
func localBin(m effect.Machine, home string) userDirs {
	return userDirs{
		{M: m, Path: home + "/.local", Mode: 0o755},
		{M: m, Path: home + "/.local/bin", Mode: 0o755},
	}
}

// program plans ~/.local/bin/ws, which the copy for root comes from, and its directories where
// missing.
func (t *Tunnel) program(self []byte) ([]step.Change, error) {
	path := t.Home + "/.local/bin/ws"
	have, err := t.M.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if err == nil && bytes.Equal(have, self) {
		return nil, nil
	}
	cs, _, err := localBin(t.M, t.Home).plan()
	if err != nil {
		return nil, err
	}
	return append(cs, Install{path, self}), nil
}

// Plan plans ws, root's copy and the Mac's key.
func (t *Tunnel) Plan(ctx context.Context) (step.Plan, error) {
	self, err := t.self()
	if err != nil {
		return step.Plan{}, fmt.Errorf("cannot read ws: %w", err)
	}
	var cs []step.Change
	copied, err := t.programAndCopy(ctx, self)
	if err != nil {
		return step.Plan{}, err
	}
	key, err := t.key(ctx)
	if err != nil {
		return step.Plan{}, err
	}
	cs = append(append(cs, copied...), key...)
	told, err := t.tell(ctx)
	if err != nil {
		return step.Plan{}, err
	}
	cs = append(cs, told...)
	if t.Conf.PublicKey != "" {
		d, err := t.daemon(ctx)
		if err != nil {
			return step.Plan{}, err
		}
		cs = append(cs, d...)
	}
	return step.Plan{Changes: cs}, nil
}

// Report says how ws, root's copy, the key and the daemon stand. A program shows no diff, and the
// key shows by its owner and mode alone: no report reads it. tunnel.pub compares with what Plan
// asked root's ws for, as a second sudo would add nothing.
func (t *Tunnel) Report(ctx context.Context) ([]step.Item, error) {
	self, err := t.self()
	if err != nil {
		return nil, fmt.Errorf("cannot read ws: %w", err)
	}
	key, err := host.Dir{M: t.M, Path: tunnelKeyFile, Mode: 0o600}.Report(ctx)
	if err != nil {
		return nil, err
	}
	for i := range key {
		key[i].Kind = step.FileItem
	}
	parts := []step.Reporter{
		localBin(t.M, t.Home),
		userFile{M: t.M, Path: t.Home + "/.local/bin/ws", Text: self, Binary: true},
		host.Dir{M: t.M, Path: "/usr/local", Mode: 0o755},
		host.Dir{M: t.M, Path: "/usr/local/libexec", Mode: 0o755},
		host.Dir{M: t.M, Path: tunnelLibexec, Mode: 0o755},
		host.File{M: t.M, Path: tunnelLibexec + "/ws", Content: self, Mode: 0o755, Secret: true},
		host.Dir{M: t.M, Path: tunnelDir, Mode: 0o700},
		host.Dir{M: t.M, Path: tunnelDir + "/routes.d", Mode: 0o755},
		fixed(key),
	}
	if t.public != "" {
		parts = append(parts, wsDirs(t.M, t.Home), t.pub())
	}
	if t.Conf.PublicKey != "" {
		daemon := step.Item{Kind: step.ServiceItem, Name: daemonLabel, Want: "loaded", Found: "not loaded"}
		if _, err := t.M.Run(ctx, "launchctl", "print", "system/"+daemonLabel); err == nil {
			daemon.Found = daemon.Want
		}
		parts = append(parts, host.File{M: t.M, Path: tunnelDir + "/tunnel.conf", Content: []byte(tunnelConf(t.Conf)), Mode: 0o600},
			host.File{M: t.M, Path: tunnelPlist, Content: t.Plist, Mode: 0o644}, fixed{daemon})
	}
	return gather(ctx, parts...)
}

func (t *Tunnel) sleep(d time.Duration) {
	if t.Sleep != nil {
		t.Sleep(d)
		return
	}
	time.Sleep(d)
}
