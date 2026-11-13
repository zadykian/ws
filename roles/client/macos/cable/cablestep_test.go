package cable_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/cable"
	"github.com/zadykian/ws/roles/client/macos/conf"
)

const cableSet = "Manual Configuration\nIP address: 10.77.0.2\nSubnet mask: 255.255.255.252\nRouter: (null)\n"

func cableStep(m *effecttest.Machine, out *effecttest.Out, list, info string) cable.Cable {
	m.Results["/usr/sbin/networksetup -listallnetworkservices"] = effecttest.Result{Out: []byte(list)}
	m.Results["/usr/sbin/networksetup -getinfo Dock"] = effecttest.Result{Out: []byte(info)}
	return cable.Cable{
		Conf: conf.Conf{CableService: "Dock"}, M: m, Out: out,
		LookPath: func(string) (string, error) { return "/usr/sbin/networksetup", nil },
	}
}

func TestCablePlans(t *testing.T) {
	const list = "header\nWi-Fi\nDock\n"
	for name, tc := range map[string]struct {
		list, info, err string
		changes         int
	}{
		"set":      {list, cableSet, "", 0},
		"dhcp":     {list, "DHCP Configuration\nIP address: 10.0.0.9\n", "", 1},
		"uplink":   {list, "Manual Configuration\nRouter: 10.0.0.1\n", "Router: 10.0.0.1", 0},
		"missing":  {"header\nWi-Fi\n", cableSet, "lacks", 0},
		"disabled": {"header\n*Dock\n", cableSet, "", 0},
	} {
		m, out := effecttest.NewMachine(), &effecttest.Out{}
		p, err := cableStep(m, out, tc.list, tc.info).Plan(context.Background())
		if tc.err != "" && (err == nil || !strings.Contains(err.Error(), tc.err)) {
			t.Errorf("%s: err = %v, want %q", name, err, tc.err)
		}
		if tc.err == "" && (err != nil || len(p.Changes) != tc.changes) {
			t.Errorf("%s: Plan = %v, %v; want %d changes", name, p.Changes, err, tc.changes)
		}
	}
}

func TestCableWithoutServiceLeavesTheNetworkAlone(t *testing.T) {
	m := effecttest.NewMachine()
	p, err := cable.Cable{M: m, Out: &effecttest.Out{}}.Plan(context.Background())
	if err != nil || len(p.Changes) != 0 || len(m.Calls) != 0 {
		t.Errorf("Plan = %v, %v, calls %v; want none", p.Changes, err, m.Calls)
	}
}

func TestCableAppliesSetmanualThroughSudo(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	s := cableStep(m, out, "h\nDock\n", "DHCP Configuration\n")
	p, _ := s.Plan(context.Background())
	m.Results["/usr/sbin/networksetup -setmanual Dock 10.77.0.2 255.255.255.252"] = effecttest.Result{}
	m.Results["/usr/sbin/networksetup -getinfo Dock"] = effecttest.Result{Out: []byte(cableSet)}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if got := step.Headline(p.Changes[0]); got != "would run networksetup -setmanual \"Dock\" 10.77.0.2 255.255.255.252" {
		t.Errorf("headline = %q", got)
	}
	if got := m.Calls[len(m.Calls)-2]; got != "root /usr/sbin/networksetup -setmanual Dock 10.77.0.2 255.255.255.252" {
		t.Errorf("last call = %q", got)
	}
	m.Results["/usr/sbin/networksetup -setmanual Dock 10.77.0.2 255.255.255.252"] = effecttest.Result{Err: errors.New("no")}
	err := s.Apply(context.Background(), p)
	if err == nil || !strings.HasPrefix(err.Error(), "would run networksetup -setmanual") {
		t.Errorf("Apply err = %v, want the command's headline first", err)
	}
}

func TestCableApplyNamesTheRemedyWhereSetmanualDoesNotTake(t *testing.T) {
	m := effecttest.NewMachine()
	s := cableStep(m, &effecttest.Out{}, "h\nDock\n", "DHCP Configuration\n")
	p, _ := s.Plan(context.Background())
	m.Results["/usr/sbin/networksetup -setmanual Dock 10.77.0.2 255.255.255.252"] = effecttest.Result{}
	err := s.Apply(context.Background(), p)
	if err == nil || !strings.Contains(err.Error(), "set it by hand in System Settings") {
		t.Errorf("Apply err = %v, want the remedy", err)
	}
}
