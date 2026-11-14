package link

import (
	"context"
	"errors"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestTunnelShake(t *testing.T) {
	for name, tc := range map[string]struct {
		server, status string
		want           bool
	}{
		"fresh": {"ws", "latest handshake: 180 s ago\n", true},
		"stale": {"ws", "latest handshake: 181 s ago\n", false},
		"none":  {"ws", "peer: x\n", false},
		"lab":   {"lab", "latest handshake: 5 s ago\n", true},
	} {
		tun, m := newTunnel(keyA, session.Status{}, false)
		tun.Server = tc.server
		m.Results[libexec+" client macos tunnel status --server "+tc.server] = effecttest.Result{Out: []byte(tc.status)}
		// With a second daemon installed, status with no --server refuses to pick one.
		m.Results[libexec+" client macos tunnel status"] = effecttest.Result{Err: errors.New("several tunnel daemons")}
		tun.Sleep = func(time.Duration) {}
		if got := tun.shake(context.Background(), 3); got != tc.want {
			t.Errorf("%s: shake = %v; want %v", name, got, tc.want)
		}
	}
}
