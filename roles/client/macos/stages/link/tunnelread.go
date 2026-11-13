package link

import (
	"context"
	"net/netip"
	"strings"

	"github.com/zadykian/ws/engine/core/check"
)

// pubFile is the Mac's public key, which ws client macos apply writes from root's key at each run.
// It needs no sudo, whose timestamp lapses during ws's run.
func (t *Tunnel) pubFile() string { return t.Home + "/.config/ws/tunnel.pub" }

// pub returns the Mac's public key from tunnel.pub, or an empty string where it holds none, as
// before ws client macos apply first wrote it.
func (t *Tunnel) pub() string {
	b, err := t.M.ReadFile(t.pubFile())
	if key := strings.TrimSpace(string(b)); err == nil && check.Key(key) {
		return key
	}
	return ""
}

// loaded tells whether launchd has ws-tunnel loaded. Anyone may read its system domain.
func (t *Tunnel) loaded(ctx context.Context) bool {
	_, err := t.M.Run(ctx, "launchctl", "print", "system/"+t.label())
	return err == nil
}

// port tells whether ws's SSH port answers at its end of the tunnel within 3 s, which takes a
// handshake. It needs no sudo and tries no login. -G bounds macOS nc's connect, and -w that of
// Homebrew's netcat, which ignores -G and comes first on the wizard's PATH.
func (t *Tunnel) port(ctx context.Context, end netip.Addr) bool {
	_, err := t.M.Run(ctx, "nc", "-z", "-G", "3", "-w", "3", end.String(), "22")
	return err == nil
}

// via tells whether ssh -G of the server gives ws's end of the tunnel, as config.d/ws does.
func (t *Tunnel) via(ctx context.Context, end netip.Addr) bool {
	out, err := t.M.Run(ctx, "ssh", "-G", t.SSH.Host)
	for _, l := range strings.Split(string(out), "\n") {
		if f := strings.Fields(l); len(f) == 2 && f[0] == "hostname" {
			return err == nil && f[1] == end.String()
		}
	}
	return false
}
