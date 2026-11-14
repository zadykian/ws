package link

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/wizard"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
)

func TestTunnelHandshakeNamesTheServerInItsMessages(t *testing.T) {
	ctx := context.Background()
	tun, m := newTunnel(keyB, session.Status{WgKey: keyB, Peered: true, MacKey: keyA}, false)
	tun.SSH.Host, tun.Server = "lab", "lab"
	tun.Hand, tun.Sleep = &wizard.Hand{}, func(time.Duration) {}
	if err := tun.handshake(ctx, conf.Conf{}, false); err != nil || !strings.Contains(strings.Join(tun.Hand.Lines(), "\n"), "so ssh lab fails") {
		t.Errorf("handshake with none = %v, closing list %q; want ssh lab", err, tun.Hand.Lines())
	}
	m.Results["ssh -G lab"] = effecttest.Result{Out: []byte("user root\nhostname 10.99.0.1\n")}
	m.Results[libexec+" client macos tunnel status --server lab"] = effecttest.Result{Out: []byte("latest handshake: 5 s ago\n")}
	m.Results[tunnelCall(tun.SSH, "-o", "BatchMode=yes", "lab", "echo", "ok")] = effecttest.Result{Out: []byte("no\n")}
	if err := tun.handshake(ctx, conf.Conf{}, false); err == nil || !strings.Contains(err.Error(), "ssh lab failed through the tunnel") {
		t.Errorf("handshake with a failed ssh = %v; want ssh lab", err)
	}
}
