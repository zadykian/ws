package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

const (
	keyA = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	keyB = "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB="
)

// newTunnel returns the stage over a Mac whose mac.conf holds WS_PUBLIC_KEY pub, with ws's status.
func newTunnel(pub, status string, check bool) (*Tunnel, *effecttest.Machine) {
	r, m := newRun("lan", status, false, check)
	m.Files["/h/mac.conf"] = append(m.Files["/h/mac.conf"], []byte("WS_PUBLIC_KEY="+pub+"\n")...)
	m.Files["/h/.config/ws/tunnel.pub"] = []byte(keyA + "\n")
	m.Results["launchctl print system/"+label] = effecttest.Result{}
	m.Results["nc -z -G 3 -w 3 10.99.0.1 22"] = effecttest.Result{}
	m.Results["ssh -G ws"] = effecttest.Result{Out: []byte("user root\nhostname 10.99.0.1\n")}
	return &Tunnel{Host: r.Host, Run: r, Home: "/h", M: m, Out: r.Host.Out}, m
}

func TestTunnelPlan(t *testing.T) {
	for name, tc := range map[string]struct{ pub, status, want string }{
		"done": {keyB, "wg0 " + keyB + "\npeer ok\nmac.pub " + keyA + "\n", ""},
		"peer": {keyB, "wg0 " + keyB + "\npeer none\nmac.pub " + keyA + "\n",
			"run ws server apply --only wireguard"},
		"other key": {keyB, "wg0 " + keyB + "\npeer ok\nmac.pub " + keyB + "\n",
			"write the Mac's key, " + keyA},
		"new wg0": {keyA, "wg0 " + keyB + "\npeer ok\nmac.pub " + keyA + "\n",
			"set WS_PUBLIC_KEY in mac.conf to wg0's key, " + keyB},
		"no wg0": {"", "peer none\n", "to wg0's key, which that run makes"},
	} {
		tun, _ := newTunnel(tc.pub, tc.status, false)
		p, err := tun.Plan(context.Background())
		var got []string
		for _, c := range p.Changes {
			got = append(got, c.String())
		}
		all := strings.Join(got, "\n")
		if err != nil || (tc.want == "") != (all == "") || !strings.Contains(all, tc.want) {
			t.Errorf("%s: Plan = %q, %v; want %q", name, all, err, tc.want)
		}
	}
}

func TestTunnelPlanWaits(t *testing.T) {
	tun, m := newTunnel(keyB, "peer ok\n", true)
	delete(m.Files, "/h/.config/ws/tunnel.pub")
	p, err := tun.Plan(context.Background())
	if err != nil || len(p.Changes) != 0 || !strings.Contains(tun.Out.(*effecttest.Out).Lines[0], "after ws mac apply") {
		t.Errorf("Plan = %v, %v; want it to wait on ws mac apply", p, err)
	}
}

func TestTunnelPlanShowsConfsDiffUnderCheck(t *testing.T) {
	for _, check := range []bool{true, false} {
		tun, m := newTunnel(keyA, "wg0 "+keyB+"\npeer ok\nmac.pub "+keyA+"\n", check)
		if _, err := tun.Plan(context.Background()); err != nil {
			t.Fatal(err)
		}
		out := strings.Join(tun.Out.(*effecttest.Out).Lines, "\n")
		shown := strings.Contains(out, "-WS_PUBLIC_KEY="+keyA+"\n+WS_PUBLIC_KEY="+keyB)
		if shown != check || !strings.Contains(string(m.Files["/h/mac.conf"]), "WS_PUBLIC_KEY="+keyA) {
			t.Errorf("check %v: diff shown %v, mac.conf %q", check, shown, m.Files["/h/mac.conf"])
		}
	}
}
