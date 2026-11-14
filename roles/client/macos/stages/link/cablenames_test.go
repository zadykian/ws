package link

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/core/wizard"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/session/sessiontest"
)

// labCable is newCable's stage with the server lab, whose lines must not name ws.
func labCable(t *testing.T, status session.Status, link string, env map[string]string) (*Cable, *effecttest.Machine, *effecttest.Prompt) {
	c, m, p := newCable(t, status, link, env)
	c.SSH.Host = "lab"
	for k, v := range m.Results {
		if n := strings.NewReplacer(" ws echo ok", " lab echo ok", "-F ws -f", "-F lab -f").Replace(k); n != k {
			m.Results[n] = v
		} else if strings.HasPrefix(k, "ssh -G") && strings.HasSuffix(k, " ws") {
			m.Results[strings.TrimSuffix(k, "ws")+"lab"] = v
		}
	}
	m.Files["/h/mac.conf"] = []byte("WS_TARGET=lan\n" + wayConf)
	return c, m, p
}

func TestCablePlanNamesTheServer(t *testing.T) {
	for name, tc := range map[string]struct {
		conf, cable string
		env         map[string]string
		want        string
	}{
		"another": {"WS_CABLE_PORT=eth1\nWS_CABLE_SERVICE=\"USB\"\n", "eth0", nil, "eth1 in /etc/ws/host_vars.yml on lab again"},
		"forced":  {"", "", map[string]string{"WS_CABLE_PORT": "eth1"}, "ask lab's port of the cable, name it in /etc/ws/host_vars.yml on lab, and"},
		"unnamed": {"", "", nil, "ask lab's port of the cable, name it in /etc/ws/host_vars.yml on lab, and"},
	} {
		c, m, _ := labCable(t, session.Status{Cable: tc.cable}, cableLink, tc.env)
		m.Files["/h/mac.conf"] = append(m.Files["/h/mac.conf"], tc.conf...)
		p, err := c.Plan(context.Background())
		var got []string
		for _, ch := range p.Changes {
			got = append(got, ch.String())
		}
		if all := strings.Join(got, "\n"); err != nil || !strings.Contains(all, tc.want) {
			t.Errorf("%s: Plan = %q, %v; want %q", name, all, err, tc.want)
		}
	}
}

func TestCableHandNamesTheServer(t *testing.T) {
	c, _, _ := labCable(t, session.Status{}, "eth0 - -\n", nil)
	c.ports = []cablePort{{"eth0", "-", "-"}}
	var h wizard.Hand
	c.Hand(context.Background(), &h)
	var b strings.Builder
	h.Close(context.Background(), &b)
	for _, want := range []string{"display that lab's cable plugs into", "eth0, which lab keeps down"} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("list %q lacks %q", b.String(), want)
		}
	}
}

func TestCableApplyRefusalsNameTheServer(t *testing.T) {
	c, m, _ := labCable(t, session.Status{}, cableLink, map[string]string{"WS_CABLE_PORT": "eth1"})
	linked(m)
	m.Results[sessiontest.Op("wait", "eth1")] = effecttest.Result{Err: context.Canceled}
	err := c.Apply(context.Background(), step.Plan{})
	if err == nil || !strings.Contains(err.Error(), "lab") || strings.Contains(strings.ReplaceAll(err.Error(), "/etc/ws/", ""), " ws") {
		t.Errorf("Apply = %v; want a refusal that names lab", err)
	}
}
