package network

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestTheStepRefuses(t *testing.T) {
	tests := []struct {
		name, iface, address string
		arrange              func(r rig)
		want                 string
	}{
		{"a bad address", "enp5s0", "10.77.0.3/30", func(rig) {}, "network_direct_link_address: 10.77.0.3 is the broadcast address of 10.77.0.0/30"},
		{"a name with a slash", "../x", "10.77.0.1/30", func(rig) {}, `"../x" is not an interface's name`},
		{"an uplink with a default route", "enp5s0", "10.77.0.1/30", func(r rig) {
			r.m.Results[ipPrefix+"-6 route show default table all"] = effecttest.Result{Out: []byte(`[{"dev":"enp5s0","table":"main"}]`)}
		}, "not an uplink, yet enp5s0 carries a default route, in table main"},
		{"a lease from DHCP", "enp5s0", "10.77.0.1/30", func(r rig) {
			r.m.Results[ipPrefix+"address show dev enp5s0"] = effecttest.Result{Out: []byte(
				`[{"ifname":"enp5s0","address":"00:1a:2b:3c:4d:5e","addr_info":[{"family":"inet","local":"192.168.1.5","prefixlen":24,"scope":"global","dynamic":true}]}]`)}
		}, "enp5s0 holds 192.168.1.5/24 from DHCP, outside 10.77.0.0/30"},
		{"a sibling entry that fits", "enp5s0", "10.77.0.1/30", func(r rig) {
			r.m.Results["netplan get --root-dir "+scratchDir] = effecttest.Result{Out: []byte("network:\n  ethernets:\n    enp5s0: {}\n    lan:\n      match:\n        name: enp*\n")}
		}, "netplan's entry lan fits enp5s0"},
		{"networkd that does not run", "enp5s0", "10.77.0.1/30", func(r rig) {
			r.u.States = nil
		}, "systemd-networkd is not running"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newRig(tt.iface, tt.address)
			tt.arrange(r)
			_, err := r.run(t, false)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want one with %q", err, tt.want)
			}
			if _, ok := r.m.Files[netplanPath]; ok {
				t.Error("the file landed")
			}
		})
	}
}

func TestCheckLeavesTheMachineAlone(t *testing.T) {
	r := newRig("enp5s0", "10.77.0.1/30")
	changed, err := r.run(t, true)
	if err != nil || !changed {
		t.Fatalf("changed = %v, err = %v", changed, err)
	}
	if slices.Contains(r.m.Calls, "root netplan generate") || slices.Contains(r.m.Calls, "write root "+netplanPath+" 600") {
		t.Errorf("--check changed the machine: %q", r.m.Calls)
	}
	if _, ok := r.m.Files[netplanPath]; ok {
		t.Error("the file landed")
	}
}
