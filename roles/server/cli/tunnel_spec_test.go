package cli

import (
	"encoding/hex"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/resource/ufw"
	"github.com/zadykian/ws/engine/core/step"
)

const (
	netdevFile  = "/etc/systemd/network/60-wg0.netdev"
	networkFile = "/etc/systemd/network/60-wg0.network"
	dropsUnit   = "/etc/systemd/system/wireguard-docker-user.service"
	macPub      = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	// tunnelPeers are the peers mac, whose key file is there, and tv, whose file is missing.
	tunnelPeers = "---\nWS_TUNNEL_PEERS:\n  - {name: mac, address: 10.99.0.2}\n  - {name: tv, address: 10.99.0.3}\n"
	// netdevBytes is the .netdev that the step rendered for the port 51820 and the peer mac alone.
	netdevBytes = `# Written by github.com/zadykian/ws, whose next run replaces any change made here.
#
# wg0, the tunnel's end on this machine. networkd reads the private key from the file below, made
# here. Each peer's public key is that of /etc/wireguard/NAME.pub, and a peer without that file is
# left out. The server has no endpoint for a peer and never speaks first: it answers each peer at
# the address its last packet came from.
[NetDev]
Name=wg0
Kind=wireguard

[WireGuard]
PrivateKeyFile=/etc/systemd/network/60-wg0.key
ListenPort=51820

# mac
[WireGuardPeer]
PublicKey=` + macPub + `
AllowedIPs=10.99.0.2/32
`
)

// tunnelMachine is a machine with the group systemd-network and mac's key.
func tunnelMachine() *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Dirs["/etc"], m.Dirs["/etc/wireguard"] = 0o755, 0o700
	// systemd makes networkd's directory: applyUnit applies one plan, which would make it twice.
	m.Dirs["/etc/systemd"], m.Dirs["/etc/systemd/network"] = 0o755, 0o755
	m.Files["/etc/group"] = []byte("root:x:0:\nsystemd-network:x:998:\n")
	m.Files["/etc/wireguard/mac.pub"] = []byte(macPub + "\n")
	return m
}

func TestTunnelSpecNetdevBytes(t *testing.T) {
	m := tunnelMachine()
	units, _ := cableUnits(t, m, tunnelPeers)
	applyUnit(t, units, "service/tunnel")
	if got := string(m.Files[netdevFile]); got != netdevBytes {
		t.Errorf("the .netdev:\n%s\nwant:\n%s", got, netdevBytes)
	}
	if got := string(m.Files[networkFile]); !strings.HasSuffix(got, "[Network]\nAddress=10.99.0.1/24\n") {
		t.Errorf("the .network:\n%s", got)
	}
	if got := applyUnit(t, units, "service/tunnel"); got != nil {
		t.Errorf("second plan %q, want none", got)
	}
}

func TestTunnelNetdevIs640OfSystemdNetwork(t *testing.T) {
	m := tunnelMachine()
	units, _ := cableUnits(t, m, tunnelPeers)
	applyUnit(t, units, "service/tunnel")
	if mode, owner := m.Modes[netdevFile], m.Owners[netdevFile]; mode != 0o640 || owner != [2]int{0, 998} {
		t.Errorf(".netdev: mode %o, owner %v; want root:998 at 640", mode, owner)
	}
	if mode, owner := m.Modes[networkFile], m.Owners[networkFile]; mode != 0o644 || owner != [2]int{} {
		t.Errorf(".network: mode %o, owner %v; want root's at 644", mode, owner)
	}
}

func TestTunnelPortRuleFollowsThePort(t *testing.T) {
	m := tunnelMachine()
	// The second rule, on wg0, and the third, for TCP, carry the comment too, as rules made by hand.
	comment := " comment=" + hex.EncodeToString([]byte("WireGuard")) + "\n"
	m.Files[ufw.UserRules] = []byte("### tuple ### allow udp 51820 0.0.0.0/0 any 0.0.0.0/0 in" + comment +
		"### tuple ### allow udp 8000 0.0.0.0/0 any 0.0.0.0/0 in_wg0" + comment +
		"### tuple ### allow tcp 8001 0.0.0.0/0 any 0.0.0.0/0 in" + comment)
	units, _ := cableUnits(t, m, "---\nWS_TUNNEL_PORT: 4000\n")
	steps := map[string]step.Step{}
	for _, u := range units {
		steps[u.Name] = u.Step
	}
	got, err := planned(t, steps, "state/tunnel-port")
	if err != nil || len(got) != 2 || !strings.Contains(got[0], "port 4000") || !strings.Contains(got[1], "delete allow") || !strings.Contains(got[1], "port 51820") {
		t.Errorf("plan = %q, %v; want 4000 added and 51820 deleted alone", got, err)
	}
}

func TestTunnelDropsNameTheCable(t *testing.T) {
	for yml, want := range map[string]string{
		"---\n": "ExecStart=/usr/local/lib/ws/wireguard/docker-user.sh wg0\n",
		"---\nWS_CABLE_PORT: enp5s0\nWS_CABLE_ADDRESS: 10.77.0.1/30\n": "ExecStart=/usr/local/lib/ws/wireguard/docker-user.sh wg0 enp5s0\n",
	} {
		m := tunnelMachine()
		units, _ := cableUnits(t, m, yml)
		applyUnit(t, units, "service/tunnel-drops")
		if got := string(m.Files[dropsUnit]); !strings.Contains(got, want) {
			t.Errorf("%q: the unit:\n%s\nwant %q", yml, got, want)
		}
		if mode := m.Modes["/usr/local/lib/ws/wireguard/docker-user.sh"]; mode != 0o755 {
			t.Errorf("the script's mode %o, want 755", mode)
		}
	}
}

func TestTunnelOrder(t *testing.T) {
	units, _ := cableUnits(t, tunnelMachine(), "---\n")
	names := kind.Names(units)
	at := func(name string) int { return slices.Index(names, name) }
	guard, key, service, wg0, port := at("guard/tunnel"), at("state/tunnel-key"), at("service/tunnel"),
		at("state/tunnel-wg0"), at("state/tunnel-port")
	if guard < 0 || key < guard || service < key || wg0 < service || port < guard {
		t.Errorf("units %q, want the guard, the key, the files, then wg0; the port after the guard", names)
	}
}

func TestTunnelGuardRunsTheCheck(t *testing.T) {
	m := tunnelMachine()
	m.Results[selfPath+" server tunnel check"] = effecttest.Result{Out: []byte("a line\n")}
	units, out := cableUnits(t, m, "---\n")
	applyUnit(t, units, "guard/tunnel")
	if !slices.Contains(m.Calls, "root "+selfPath+" server tunnel check") || !slices.Contains(out.Lines, "say: a line") {
		t.Errorf("calls %q, said %q; want ws's check as root, and its line", m.Calls, out.Lines)
	}
}
