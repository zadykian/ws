package link

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/session/sessiontest"
)

func TestCableApplyNamesTheServer(t *testing.T) {
	c, m, p := labCable(t, session.Status{}, cableLink, nil)
	linked(m)
	p.Answers["lab's port of the cable, of eth0, eth1 (carrier) (WS_CABLE_PORT) [eth1, or none]"] = "eth1"
	p.Answers["The Mac's service of the cable, of Wi-Fi, USB LAN (WS_CABLE_SERVICE) [USB LAN, or none]"] = "USB LAN"
	if err := c.Apply(context.Background(), step.Plan{}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(c.Out.(*effecttest.Out).Lines, "\n"); !strings.Contains(got, "eth1 goes in /etc/ws/host_vars.yml on lab, and") {
		t.Errorf("said %q; want lab for the file's machine", got)
	}
}

func TestCableApplyLacksEndNamesTheServer(t *testing.T) {
	c, m, _ := labCable(t, session.Status{}, cableLink, map[string]string{"WS_CABLE_PORT": "eth1"})
	linked(m)
	m.Results[sessiontest.Op("wait", "eth1")] = effecttest.Result{Out: []byte("eth1 1 -\n")}
	if err := c.Apply(context.Background(), step.Plan{}); err == nil || !strings.Contains(err.Error(), "Cable: lab's eth1 lacks") {
		t.Errorf("Apply = %v; want lab's eth1", err)
	}
}

func TestCableApplyPingNamesTheServer(t *testing.T) {
	c, m, _ := labCable(t, session.Status{}, cableLink, map[string]string{"WS_CABLE_PORT": "eth1", "WS_CABLE_SERVICE": "USB LAN"})
	linked(m)
	m.Results["ping -c 1 -t 3 10.77.0.1"] = effecttest.Result{Err: context.Canceled}
	if err := c.Apply(context.Background(), step.Plan{}); err == nil || !strings.Contains(err.Error(), "10.77.0.1, lab's end of the cable") {
		t.Errorf("Apply = %v; want lab's end of the cable", err)
	}
}
