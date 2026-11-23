package cable

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/server/values"
)

// newFile is the netplan file that the file unit checks, beside the live one.
const newFile = "/etc/netplan/60-cable.yaml.ws-check"

// cableOf is the cable on the port enp5s0, at the default address.
var cableOf = values.Cable{Port: "enp5s0", Address: "10.77.0.1/30"}

func TestCableCheckRefusesTheUplink(t *testing.T) {
	for name, tc := range map[string]struct {
		result, line string
		want         string
	}{
		"a default route": {ipPrefix + "-6 route show default table all", `[{"dev":"enp5s0","table":"main"}]`,
			"not an uplink, yet enp5s0 carries a default route, in table main"},
		"a lease from DHCP": {ipPrefix + "address show dev enp5s0",
			`[{"ifname":"enp5s0","address":"00:1a:2b:3c:4d:5e","addr_info":[{"family":"inet","local":"192.168.1.5","prefixlen":24,"scope":"global","dynamic":true}]}]`,
			"enp5s0 holds 192.168.1.5/24 from DHCP, outside 10.77.0.0/30"},
	} {
		m, u := portMachine()
		m.Results[tc.result] = effecttest.Result{Out: []byte(tc.line)}
		if err := Check(t.Context(), m, u, cableOf, ""); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: check = %v, want %q", name, err, tc.want)
		}
	}
}

func TestCableCheckRefusesWhatNetplanRefuses(t *testing.T) {
	m, u := portMachine()
	m.Files[newFile] = []byte("network: {}\n")
	m.Results["netplan get --root-dir "+scratchDir] = effecttest.Result{Out: []byte("network:\n  ethernets:\n    enp5s0: {}\n    lan:\n      match:\n        name: enp*\n")}
	if err := Check(t.Context(), m, u, cableOf, newFile); err == nil || !strings.Contains(err.Error(), "netplan's entry lan fits enp5s0") {
		t.Errorf("check = %v, want the sibling refused", err)
	}
	if err := Check(t.Context(), m, u, cableOf, ""); err != nil {
		t.Errorf("without a file: check = %v, want the port alone checked", err)
	}
}

func TestCableCheckReadsTheFileAsRoot(t *testing.T) {
	m, u := portMachine()
	m.Files[newFile] = []byte("the new file\n")
	if err := Check(t.Context(), m, u, cableOf, newFile); err != nil {
		t.Fatalf("check = %v, want none", err)
	}
	read := slices.Index(m.Calls, "read root "+newFile)
	wrote := slices.Index(m.Calls, "write root "+scratchDir+"/etc/netplan/60-cable.yaml 600")
	if read < 0 || wrote < read {
		t.Errorf("calls %q, want the file read as root, then written as the cable's in the scratch root", m.Calls)
	}
}

func TestCableCheckWantsNetworkd(t *testing.T) {
	m, u := portMachine()
	u.States = nil
	if err := Check(t.Context(), m, u, cableOf, ""); err == nil || !strings.Contains(err.Error(), "systemd-networkd is not running") {
		t.Errorf("check = %v, want networkd's refusal", err)
	}
}

func TestCableCheckWithoutAPortOrItsInterface(t *testing.T) {
	m, u := portMachine()
	if err := Check(t.Context(), m, u, values.Cable{Address: "10.77.0.1/30"}, ""); err != nil || len(m.Calls) > 0 {
		t.Errorf("no port: check = %v, calls %q; want none", err, m.Calls)
	}
	if err := Check(t.Context(), m, u, values.Cable{Port: "enp9s0", Address: "10.77.0.1/30"}, ""); err == nil ||
		!strings.Contains(err.Error(), "this machine has no interface enp9s0") {
		t.Errorf("no interface: check = %v, want the port named", err)
	}
}
