package network

import (
	"slices"
	"strings"
	"testing"
)

var cable = Port{Name: "enp5s0", Altnames: []string{"enx001122334455"}, MAC: "00:1a:2b:3c:4d:5e", Driver: "r8169"}

func ok() Result { return Result{} }

func settingsOf(yaml string) Result { return Result{Stdout: yaml} }

func TestNetplanProblems(t *testing.T) {
	const own = "network:\n  version: 2\n  ethernets:\n    enp5s0:\n      addresses: [10.77.0.1/30]\n"
	tests := []struct {
		name string
		yaml string
		want []string
	}{
		{"the port alone", own, nil},
		{"no settings", "", nil},
		{"the default renderer", "network:\n  renderer: networkd\n  ethernets:\n    enp5s0: {}\n", nil},
		{"a global renderer", "network:\n  renderer: NetworkManager\n  ethernets:\n    enp5s0: {}\n",
			[]string{"netplan hands enp5s0 to NetworkManager, not networkd, which the role sets up the direct link with."}},
		{"the port's renderer", "network:\n  ethernets:\n    enp5s0:\n      renderer: NetworkManager\n",
			[]string{"netplan hands enp5s0 to NetworkManager, not networkd, which the role sets up the direct link with."}},
		{"the port's renderer wins", "network:\n  renderer: NetworkManager\n  ethernets:\n    enp5s0:\n      renderer: networkd\n", nil},
		{"a match by MAC address", own + "      match:\n        macaddress: 00:1a:2b:3c:4d:5e\n", nil},
		{"a match by MAC address in capitals", own + "      match:\n        macaddress: 00:1A:2B:3C:4D:5E\n", nil},
		{"a match by a plain name", own + "      match:\n        name: enp5s0\n", nil},
		{"a match by a glob", own + "      match:\n        name: enp*\n",
			[]string{"netplan's entry enp5s0 has a match: that may fit other interfaces too, whose DHCP the direct link would turn off: match enp5s0's MAC address, or a name without a glob."}},
		{"a match by a list of names", own + "      match:\n        name: [enp5s0]\n",
			[]string{"netplan's entry enp5s0 has a match: that may fit other interfaces too, whose DHCP the direct link would turn off: match enp5s0's MAC address, or a name without a glob."}},
		{"a match by driver alone", own + "      match:\n        driver: r8169\n",
			[]string{"netplan's entry enp5s0 has a match: that may fit other interfaces too, whose DHCP the direct link would turn off: match enp5s0's MAC address, or a name without a glob."}},
		{"a match that misses the port", own + "      match:\n        macaddress: aa:bb:cc:dd:ee:ff\n",
			[]string{"netplan's entry enp5s0 has a match: that enp5s0 doesn't fit, so the direct link would go to another interface."}},
		{"a match by another driver", own + "      match:\n        driver: e1000e\n",
			[]string{"netplan's entry enp5s0 has a match: that enp5s0 doesn't fit, so the direct link would go to another interface."}},
		{"a negated driver", own + "      match:\n        macaddress: 00:1a:2b:3c:4d:5e\n        driver: '!r8169'\n",
			[]string{"netplan's entry enp5s0 has a match: that enp5s0 doesn't fit, so the direct link would go to another interface."}},
		{"another entry", own + "    eth9:\n      dhcp4: true\n", nil},
		{"another entry by the port's alternative name", own + "    enx001122334455:\n      dhcp4: true\n",
			[]string{"netplan's entry enx001122334455 fits enp5s0, and keeps a networkd file apart from the direct link's: move its settings to enp5s0, or narrow its match: to another interface."}},
		{"another entry with a match that fits", own + "    lan:\n      match:\n        name: 'enp*'\n",
			[]string{"netplan's entry lan fits enp5s0, and keeps a networkd file apart from the direct link's: move its settings to enp5s0, or narrow its match: to another interface."}},
		{"another entry with an empty match", own + "    lan:\n      match: {}\n",
			[]string{"netplan's entry lan fits enp5s0, and keeps a networkd file apart from the direct link's: move its settings to enp5s0, or narrow its match: to another interface."}},
		{"another entry with a match that misses", own + "    lan:\n      match:\n        name: 'eth*'\n", nil},
		{"another entry that is not a map", own + "    odd: 3\n", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NetplanProblems(ok(), settingsOf(tt.yaml), "/tmp/root", cable); !slices.Equal(got, tt.want) {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNetplanProblemsRefusals(t *testing.T) {
	const root = "/tmp/root"
	tests := []struct {
		name           string
		generated, got Result
		want, never    string
	}{
		{"netplan rejects its files", Result{RC: 1, Stderr: "Command failed: bad key.\n  password: hunter2"}, ok(),
			"netplan rejects its files with the direct link's: bad key.", "hunter2"},
		{"netplan can't read its settings", ok(), Result{RC: 1, Stderr: "ERROR: nope\n  psk: hunter2"},
			"netplan can't read its settings: ERROR: nope.", "hunter2"},
		{"netplan failed but exited with 0", ok(), Result{Stderr: "Command failed: broken"},
			"netplan can't read its settings: broken.", ""},
		{"settings that aren't YAML", ok(), Result{Stdout: "network: [unclosed"},
			"netplan get printed settings that aren't YAML.", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NetplanProblems(tt.generated, tt.got, root, cable)
			if len(got) != 1 || got[0] != tt.want {
				t.Fatalf("got %q, want [%q]", got, tt.want)
			}
			if tt.never != "" && strings.Contains(got[0], tt.never) {
				t.Errorf("the message quotes %q", tt.never)
			}
		})
	}
}
