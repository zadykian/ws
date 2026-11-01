package wizard

import (
	"context"
	"strings"

	"github.com/zadykian/ws/mac/internal/check"
)

// pubFile is the Mac's public key, which ws mac apply writes from root's key at each run. It needs no
// sudo, whose timestamp lapses during ws's run.
func (t *Tunnel) pubFile() string { return t.Home + "/.config/ws/tunnel.pub" }

// pub returns the Mac's public key from tunnel.pub, or an empty string where it holds none, as
// before ws mac apply first wrote it.
func (t *Tunnel) pub() string {
	b, err := t.M.ReadFile(t.pubFile())
	if key := strings.TrimSpace(string(b)); err == nil && check.Key(key) {
		return key
	}
	return ""
}

// loaded tells whether launchd has ws-tunnel loaded. Anyone may read its system domain.
func (t *Tunnel) loaded(ctx context.Context) bool {
	_, err := t.M.Run(ctx, "launchctl", "print", "system/"+label)
	return err == nil
}

// port tells whether ws's SSH port answers at its end of the tunnel within 3 s, which takes a
// handshake. It needs no sudo and tries no login. -G bounds macOS nc's connect, and -w that of
// Homebrew's netcat, which ignores -G and comes first on the wizard's PATH.
func (t *Tunnel) port(ctx context.Context) bool {
	_, err := t.M.Run(ctx, "nc", "-z", "-G", "3", "-w", "3", wsEnd, "22")
	return err == nil
}

// via tells whether ssh -G ws gives ws's end of the tunnel, as config.d/ws does once launchd has
// ws-tunnel.
func (t *Tunnel) via(ctx context.Context) bool {
	out, err := t.M.Run(ctx, "ssh", "-G", "ws")
	for _, l := range strings.Split(string(out), "\n") {
		if f := strings.Fields(l); len(f) == 2 && f[0] == "hostname" {
			return err == nil && f[1] == wsEnd
		}
	}
	return false
}
