package wizard

import (
	"context"
	"testing"
	"time"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
)

func TestTunnelShake(t *testing.T) {
	for name, tc := range map[string]struct {
		status string
		want   bool
	}{
		"fresh": {"latest handshake: 180 s ago\n", true},
		"stale": {"latest handshake: 181 s ago\n", false},
		"none":  {"peer: x\n", false},
	} {
		tun, m := newTunnel(keyA, "", false)
		m.Results[libexec+" tunnel status"] = effecttest.Result{Out: []byte(tc.status)}
		tun.Sleep = func(time.Duration) {}
		if got := tun.shake(context.Background(), 3); got != tc.want {
			t.Errorf("%s: shake = %v; want %v", name, got, tc.want)
		}
	}
}
