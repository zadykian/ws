package link

import (
	"context"
	"errors"
	"github.com/zadykian/ws/engine/core/effect/local"
	"github.com/zadykian/ws/engine/core/remote"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/stages/host"
	"github.com/zadykian/ws/roles/client/macos/stages/tools"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/values"
)

const (
	keyA = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	keyB = "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB="
)

// daemonLabel is the launchd label of ws's daemon.
const daemonLabel = "com.github.zadykian.ws-tunnel.ws"

// newTunnel returns the stage over a Mac whose values.yml holds WS_PUBLIC_KEY pub, with ws's status.
func newTunnel(pub string, status session.Status, check bool) (*Tunnel, *effecttest.Machine) {
	r, m := newRun("lan", status, false, check)
	m.Files["/h/values.yml"] = append(m.Files["/h/values.yml"], []byte(values.Text("WS_PUBLIC_KEY", pub))...)
	m.Files["/etc/ws/tunnel.pub"] = []byte(keyA + "\n")
	m.Results["launchctl print system/"+daemonLabel] = effecttest.Result{}
	m.Results["nc -z -G 3 -w 3 10.99.0.1 22"] = effecttest.Result{}
	m.Results["ssh -G ws"] = effecttest.Result{Out: []byte("user root\nhostname 10.99.0.1\n")}
	return &Tunnel{SSH: remote.SSH{Host: "ws", M: m, Lock: "/l", Ways: host.MacWays{M: m, Conf: "/h/values.yml"}}, Ws: r.Ws, Conf: "/h/values.yml", Check: check, Server: "ws", Home: "/h", M: m, Out: r.Out}, m
}

func TestStagesGiveTheWordsOfNoTicket(t *testing.T) {
	m := local.OSMachine{Ticket: func() error { return local.ErrNoTicket }}
	_, planned := (&tools.Setup{M: m, LookPath: func(string) (string, error) { return "", nil }}).Plan(context.Background())
	tun, _ := newTunnel(keyA, session.Status{}, false)
	tun.M = m
	if err := tun.Apply(context.Background(), step.Plan{}); !errors.Is(err, local.ErrNoTicket) || !errors.Is(planned, err) {
		t.Errorf("Apply = %v, Plan = %v", err, planned)
	}
}

func TestTunnelPlan(t *testing.T) {
	for name, tc := range map[string]struct {
		pub    string
		status session.Status
		want   string
	}{
		"done":      {keyB, session.Status{WgKey: keyB, Peered: true, MacKey: keyA}, ""},
		"peer":      {keyB, session.Status{WgKey: keyB, MacKey: keyA}, "run ws server ubuntu apply --only tunnel"},
		"other key": {keyB, session.Status{WgKey: keyB, Peered: true, MacKey: keyB}, "write the Mac's key, " + keyA},
		"new wg0": {keyA, session.Status{WgKey: keyB, Peered: true, MacKey: keyA},
			"set WS_PUBLIC_KEY in the server's values.yml to wg0's key, " + keyB},
		"no wg0": {"", session.Status{}, "to wg0's key, which that run makes"},
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
	tun, m := newTunnel(keyB, session.Status{Peered: true}, true)
	delete(m.Files, "/etc/ws/tunnel.pub")
	p, err := tun.Plan(context.Background())
	if err != nil || len(p.Changes) != 0 || !strings.Contains(tun.Out.(*effecttest.Out).Lines[0], "after ws client macos apply") {
		t.Errorf("Plan = %v, %v; want it to wait on ws client macos apply", p, err)
	}
}

func TestTunnelPlanShowsConfsDiffUnderCheck(t *testing.T) {
	for _, check := range []bool{true, false} {
		tun, m := newTunnel(keyA, session.Status{WgKey: keyB, Peered: true, MacKey: keyA}, check)
		if _, err := tun.Plan(context.Background()); err != nil {
			t.Fatal(err)
		}
		out := strings.Join(tun.Out.(*effecttest.Out).Lines, "\n")
		shown := strings.Contains(out, "-"+values.Line("WS_PUBLIC_KEY", keyA)+"\n+"+values.Line("WS_PUBLIC_KEY", keyB))
		if shown != check || !strings.Contains(string(m.Files["/h/values.yml"]), values.Line("WS_PUBLIC_KEY", keyA)) {
			t.Errorf("check %v: diff shown %v, values.yml %q", check, shown, m.Files["/h/values.yml"])
		}
	}
}

func TestTunnelPlanReachesWsAtItsSlot(t *testing.T) {
	tun, m := newTunnel(keyB, session.Status{WgKey: keyB, Peered: true, MacKey: keyA}, false)
	m.Files["/h/values.yml"] = append(m.Files["/h/values.yml"], values.Text("WS_SLOT", "1")...)
	m.Results["nc -z -G 3 -w 3 10.99.1.1 22"] = effecttest.Result{}
	m.Results["ssh -G ws"] = effecttest.Result{Out: []byte("user root\nhostname 10.99.1.1\n")}
	delete(m.Results, "nc -z -G 3 -w 3 10.99.0.1 22")
	p, err := tun.Plan(context.Background())
	if err != nil || len(p.Changes) != 0 {
		t.Errorf("Plan = %v, %v; want it done at slot 1's end", p, err)
	}
	m.Results["ssh -G ws"] = effecttest.Result{Out: []byte("user root\nhostname 10.99.0.1\n")}
	if p, _ := tun.Plan(context.Background()); len(p.Changes) == 0 {
		t.Error("Plan is empty where ssh -G ws gives slot 0's end")
	}
}

func TestTunnelPlanNamesTheServer(t *testing.T) {
	tun, _ := newTunnel(keyA, session.Status{WgKey: keyB, MacKey: keyB}, false)
	tun.SSH.Host, tun.Server = "lab", "lab"
	p, err := tun.Plan(context.Background())
	var got []string
	for _, c := range p.Changes {
		got = append(got, c.String())
	}
	all := strings.Join(got, "\n")
	for _, w := range []string{"/etc/wireguard/mac.pub on lab", "--only tunnel on lab,", "log in to lab through it"} {
		if err != nil || !strings.Contains(all, w) {
			t.Errorf("Plan = %q, %v; want %q", all, err, w)
		}
	}
}
