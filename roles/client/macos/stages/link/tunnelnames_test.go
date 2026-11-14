package link

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/core/wizard"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
)

func TestTunnelApplyNamesTheServer(t *testing.T) {
	tun, m := newTunnel(keyA, session.Status{}, false)
	tun.SSH.Host, tun.Server = "lab", "lab"
	m.Results["sudo -v"] = effecttest.Result{}
	m.Results[libexec+" client macos tunnel key"] = effecttest.Result{Out: []byte(keyA + "\n")}
	_ = tun.Apply(context.Background(), step.Plan{})
	if got := strings.Join(tun.Out.(*effecttest.Out).Lines, "\n"); !strings.Contains(got, "mac.pub on lab, and ws server ubuntu apply") {
		t.Errorf("said %q; want lab for the key's machine", got)
	}
	tun, m = newTunnel(keyA, session.Status{Peered: true, MacKey: keyA}, false)
	tun.SSH.Host, tun.Server = "lab", "lab"
	m.Results["sudo -v"] = effecttest.Result{}
	m.Results[libexec+" client macos tunnel key"] = effecttest.Result{Out: []byte(keyA + "\n")}
	if err := tun.Apply(context.Background(), step.Plan{}); err == nil || !strings.Contains(err.Error(), "Tunnel: lab has no wg0 key") {
		t.Errorf("Apply with no wg0 key = %v; want lab", err)
	}
}

func TestTunnelKickNamesTheServer(t *testing.T) {
	tun, m := newTunnel(keyB, session.Status{WgKey: keyB, Peered: true, MacKey: keyA}, false)
	tun.SSH.Host, tun.Server = "lab", "lab"
	m.Results["launchctl kickstart -k system/"+tun.label()] = effecttest.Result{}
	tun.Hand, tun.Sleep = &wizard.Hand{}, func(time.Duration) {}
	_ = tun.handshake(context.Background(), conf.Conf{}, true)
	if got := strings.Join(tun.Out.(*effecttest.Out).Lines, "\n"); !strings.Contains(got, "wg0's new peer on lab") {
		t.Errorf("said %q; want lab", got)
	}
}
