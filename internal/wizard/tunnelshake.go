package wizard

import (
	"context"
	"regexp"
	"strconv"
	"time"
)

// libexec is where ws mac apply puts the ws binary that root runs for the tunnel.
const libexec = "/usr/local/libexec/ws-tunnel/ws"

var age = regexp.MustCompile(`(?m)^latest handshake: (\d+) s ago$`)

// shake tells whether ws mac tunnel status shows a handshake within 180 s, past which WireGuard drops
// the session. It reads up to tries times, a second apart, as a switch of the tunnel's way removes
// the peer and adds it again. sudo asks for no password.
func (t *Tunnel) shake(ctx context.Context, tries int) bool {
	for try := 1; ; try++ {
		out, _ := t.M.RunRoot(ctx, libexec, "client", "macos", "tunnel", "status")
		if m := age.FindSubmatch(out); m != nil {
			if n, _ := strconv.Atoi(string(m[1])); n <= 180 {
				return true
			}
		}
		if try >= tries || ctx.Err() != nil {
			return false
		}
		t.Sleep(time.Second)
	}
}
