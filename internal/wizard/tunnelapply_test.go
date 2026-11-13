package wizard

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

func TestTunnelApply(t *testing.T) {
	for name, tc := range map[string]struct{ shake, hand string }{
		"handshake": {"latest handshake: 5 s ago\n", ""},
		"none":      {"", "ws-tunnel had none with ws within 30 s"},
	} {
		tun, m := newTunnel(keyA, wsStatus{WgKey: keyB, Peered: true, MacKey: keyA}, false)
		m.Results["sudo -v"] = effecttest.Result{}
		m.Results[libexec+" client macos tunnel key"] = effecttest.Result{Out: []byte(keyA + "\n")}
		if tc.shake != "" {
			m.Results[libexec+" client macos tunnel status"] = effecttest.Result{Out: []byte(tc.shake)}
		}
		m.Results["mkdir -p /h"] = effecttest.Result{}
		m.Results[tunnelCall(tun.SSH, "-o", "BatchMode=yes", "ws", "echo", "ok")] =
			effecttest.Result{Out: []byte("ok\n")}
		ran := false
		tun.MacApply = func(context.Context, bool) (bool, error) { ran = true; return true, nil }
		tun.Hand, tun.Sleep = &Hand{}, func(time.Duration) {}
		if err := tun.Apply(context.Background(), step.Plan{}); err != nil {
			t.Fatalf("%s: Apply = %v", name, err)
		}
		if got := string(m.Files["/h/mac.conf"]); !strings.Contains(got, "WS_PUBLIC_KEY="+keyB+"\n") || !ran {
			t.Errorf("%s: mac.conf = %q, setup ran %v; want wg0's key and a run", name, got, ran)
		}
		if got := strings.Join(tun.Hand.Lines(), "\n"); !strings.Contains(got, tc.hand) || (tc.hand != "") != tun.none {
			t.Errorf("%s: closing list = %q, none = %v", name, got, tun.none)
		}
	}
}
func TestTunnelApplyStops(t *testing.T) {
	tun, m := newTunnel(keyA, wsStatus{Peered: true}, false)
	m.Results["sudo -v"] = effecttest.Result{}
	m.Results[libexec+" client macos tunnel key"] = effecttest.Result{Out: []byte("x\n")}
	if err := tun.Apply(context.Background(), step.Plan{}); err == nil || !strings.Contains(err.Error(), "prints no key") {
		t.Errorf("Apply with no key = %v", err)
	}
}
func TestTunnelPlanLoaded(t *testing.T) {
	tun, m := newTunnel(keyB, wsStatus{WgKey: keyB, Peered: true, MacKey: keyA}, false)
	m.Results["launchctl print system/"+label] = effecttest.Result{Err: context.Canceled}
	if p, _ := tun.Plan(context.Background()); len(p.Changes) == 0 {
		t.Error("Plan without ws-tunnel in launchd is empty")
	}
	tun.none = true
	if p, _ := tun.Plan(context.Background()); len(p.Changes) == 0 {
		t.Error("none does not excuse the missing daemon")
	}
}
