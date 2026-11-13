package wizard

import (
	"context"
	guided "github.com/zadykian/ws/engine/core/wizard"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/session/sessiontest"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

// idleRun is ws's status after a full run of this ws, v1.2.3, that exited 0.
var idleRun = session.Status{Run: session.Stamp{Version: "v1.2.3", Exit: "0"}}

const cableLink = "eth0 - -\neth1 1 -\n"

// newCable returns the stage after a full run of this ws, whose status names the cable that status
// does.
func newCable(t *testing.T, status session.Status, link string, env map[string]string) (*Cable, *effecttest.Machine, *effecttest.Prompt) {
	root, m := newLogin(effecttest.Result{Out: []byte("ok\n")})
	status.Run = idleRun.Run
	out := &effecttest.Out{}
	ws := macSession(m, root.SSH, out, status)
	m.Results["mkdir -p /h"] = effecttest.Result{}
	m.Results[sessiontest.Op("link")] = effecttest.Result{Out: []byte(link)}
	p := &effecttest.Prompt{T: t, Answers: map[string]string{}}
	c := &Cable{Ws: ws, SSH: root.SSH, Conf: "/h/mac.conf",
		RunDone:  func(context.Context, session.Want) bool { return true },
		MacApply: func(context.Context, bool) (bool, error) { return false, nil },
		Env:      func(k string) (string, bool) { v, ok := env[k]; return v, ok }, P: p, Tty: true, M: m, Out: out}
	return c, m, p
}

func TestCablePlan(t *testing.T) {
	for name, tc := range map[string]struct {
		conf, cable, link, want string
		env                     map[string]string
	}{
		"done":       {"WS_CABLE_SERVICE=\n", "", cableLink, "", nil},
		"named":      {"WS_CABLE_PORT=eth1\nWS_CABLE_SERVICE=\"USB\"\n", "eth1", cableLink, "", nil},
		"another":    {"WS_CABLE_PORT=eth1\nWS_CABLE_SERVICE=\"USB\"\n", "eth0", cableLink, "again", nil},
		"undocked":   {"", "", "eth0 - -\n", "", nil},
		"forced":     {"", "", "eth0 - -\n", "ask ws's port", map[string]string{"WS_CABLE_PORT": "eth0"}},
		"docked":     {"", "", cableLink, "ask ws's port", nil},
		"has end":    {"", "", "eth1 1 10.77.0.1/24\n", "ask the Mac's service", nil},
		"none given": {"", "", cableLink, "empty WS_CABLE_SERVICE", map[string]string{"WS_CABLE_SERVICE": ""}},
	} {
		c, m, _ := newCable(t, session.Status{Cable: tc.cable}, tc.link, tc.env)
		m.Files["/h/mac.conf"] = []byte("WS_TARGET=lan\n" + wayConf + tc.conf)
		p, err := c.Plan(context.Background())
		got := errText(err)
		for _, ch := range p.Changes {
			got += ch.String()
		}
		if !strings.Contains(got, tc.want) || (tc.want == "" && got != "") {
			t.Errorf("%s: Plan = %q; want %q", name, got, tc.want)
		}
	}
}

func TestCableApplyAsksPortAndService(t *testing.T) {
	c, m, p := newCable(t, session.Status{}, cableLink, nil)
	m.Results[sessiontest.Op("link", "eth1")] = effecttest.Result{}
	m.Results[sessiontest.Op("run", "--only", "network,wireguard")] = effecttest.Result{}
	m.Results[sessiontest.Op("wait", "eth1")] = effecttest.Result{Out: []byte("eth1 1 10.77.0.1/24\n")}
	m.Results["networksetup -listallnetworkservices"] = effecttest.Result{Out: []byte("An asterisk (*) denotes that a network service is disabled.\nWi-Fi\nUSB LAN\n")}
	m.Results["networksetup -getinfo Wi-Fi"] = effecttest.Result{Out: []byte("IP address: 10.0.0.2\n")}
	m.Results["networksetup -getinfo USB LAN"] = effecttest.Result{Out: []byte("IP address: 169.254.3.4\n")}
	m.Results["ping -c 1 -t 3 10.77.0.1"] = effecttest.Result{}
	p.Answers["ws's port of the cable, of eth0, eth1 (carrier) (WS_CABLE_PORT) [eth1, or none]"] = "eth1"
	p.Answers["The Mac's service of the cable, of Wi-Fi, USB LAN (WS_CABLE_SERVICE) [USB LAN, or none]"] = "USB LAN"
	if err := c.Apply(context.Background(), step.Plan{}); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/h/mac.conf"]); !strings.Contains(got, "WS_CABLE_PORT=eth1\nWS_CABLE_SERVICE=\"USB LAN\"\n") {
		t.Errorf("mac.conf = %q; want the port and the service", got)
	}
}

func TestCableApplyNone(t *testing.T) {
	c, m, p := newCable(t, session.Status{}, cableLink, nil)
	p.Answers["ws's port of the cable, of eth0, eth1 (carrier) (WS_CABLE_PORT) [eth1, or none]"] = "none"
	if err := c.Apply(context.Background(), step.Plan{}); err != nil || !strings.Contains(string(m.Files["/h/mac.conf"]), "WS_CABLE_SERVICE=\n") {
		t.Errorf("Apply with none = %v, mac.conf %q; want an empty service", err, m.Files["/h/mac.conf"])
	}
}

func TestCableApplyEnvPortOutsideList(t *testing.T) {
	c, _, _ := newCable(t, session.Status{}, cableLink, map[string]string{"WS_CABLE_PORT": "eth9"})
	if err := c.Apply(context.Background(), step.Plan{}); err == nil || !strings.Contains(err.Error(), "none of eth0, eth1") {
		t.Errorf("Apply = %v; want a refusal of the environment's port", err)
	}
}

func TestCableHand(t *testing.T) {
	c, _, _ := newCable(t, session.Status{}, "eth0 - -\n", nil)
	c.ports = []cablePort{{"eth0", "-", "-"}}
	var h guided.Hand
	c.Hand(context.Background(), &h)
	var b strings.Builder
	h.Close(context.Background(), &b)
	for _, want := range []string{"dock the Mac", "plugs into eth0, which ws keeps down", "WS_CABLE_SERVICE="} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("list %q lacks %q", b.String(), want)
		}
	}
}

func TestCablePlanAfterRunUnderCheck(t *testing.T) {
	c, m, _ := newCable(t, session.Status{}, cableLink, nil)
	m.Files["/h/mac.conf"] = []byte("WS_TARGET=lan\n" + wayConf)
	c.Check = true
	c.RunDone = func(context.Context, session.Want) bool { return false }
	p, err := c.Plan(context.Background())
	out := strings.Join(c.Out.(*effecttest.Out).Lines, "\n")
	if err != nil || len(p.Changes) != 0 || out != "say: Cable: after ws's run" {
		t.Errorf("Plan = %v, %v, said %q; want the stage after ws's run", p, err, out)
	}
}

func TestCablePlanAfterRunWithNoWayUnderCheck(t *testing.T) {
	c, m, _ := newCable(t, session.Status{}, cableLink, nil)
	m.Files["/h/mac.conf"] = []byte("WS_TARGET=lan\n" + wayConf)
	c.Check = true
	for k := range m.Results {
		if strings.HasPrefix(k, "ssh -G") {
			delete(m.Results, k)
		}
	}
	p, err := c.Plan(context.Background())
	out := strings.Join(c.Out.(*effecttest.Out).Lines, "\n")
	if err != nil || len(p.Changes) != 0 || out != "say: Cable: after ws's run" {
		t.Errorf("Plan = %v, %v, said %q; want the stage after ws's run", p, err, out)
	}
}

func TestCablePlanReadsNoPortsWhereTheEnvironmentNamesOne(t *testing.T) {
	c, m, _ := newCable(t, session.Status{}, cableLink, map[string]string{"WS_CABLE_PORT": "eth1"})
	m.Files["/h/mac.conf"] = []byte("WS_TARGET=lan\n" + wayConf)
	if _, err := c.Plan(context.Background()); err != nil || len(c.ports) != 0 {
		t.Errorf("Plan = %v, read ports %v; want no read of link", err, c.ports)
	}
}
