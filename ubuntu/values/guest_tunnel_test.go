package values

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

const labAndBuild = `WS_INCUS_GUESTS:
  - {name: lab, address: 10.100.0.10, cpu: 4, memory: 8GiB, disk: 200GiB}
  - {name: build, address: 10.100.0.11, cpu: 2, memory: 4GiB, disk: 100GiB}
`

// guestMachine holds each guest's own values file.
func guestMachine(files map[string]string) *effecttest.Machine {
	m := effecttest.NewMachine()
	for name, text := range files {
		m.Files["/etc/ws/guests/"+name+"/values.yml"] = []byte(text)
	}
	return m
}

func hostGuests(t *testing.T) Host {
	h, err := ParseHost([]byte(labAndBuild))
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestTheHostHoldsATunnelOfItsOwn(t *testing.T) {
	h, err := ParseHost([]byte("WS_TUNNEL_ADDRESS: 10.99.3.1/24\nWS_TUNNEL_PORT: 51830\nWS_TUNNEL_PEERS: []\n"))
	if err != nil || h.Tunnel.Port != 51830 || len(h.Tunnel.Peers) != 0 {
		t.Errorf("tunnel = %+v, err = %v", h.Tunnel, err)
	}
	if d, err := ParseHost(nil); err != nil || len(d.Tunnel.Peers) != 1 || d.Tunnel.Peers[0].Name != "mac" {
		t.Errorf("default peers = %+v, err = %v, want the Mac", d.Tunnel.Peers, err)
	}
	if _, err := ParseHost([]byte("WS_TUNNEL_PORT: 0\n")); err == nil {
		t.Error("a port 0 passed")
	}
}

func TestEachGuestHoldsATunnelAtItsSlotAndPort(t *testing.T) {
	h := hostGuests(t)
	m := guestMachine(map[string]string{
		"lab":   "WS_TUNNEL_ADDRESS: 10.99.1.1/24\nWS_TUNNEL_PORT: 51821\nWS_IKEWARD_FORWARD_FROM: [10.99.1.2]\n",
		"build": "WS_TUNNEL_ADDRESS: 10.99.2.1/24\nWS_TUNNEL_PORT: 51822\nWS_TUNNEL_PEERS:\n  - {name: mac, address: 10.99.2.2}\n  - {name: tv, address: 10.99.2.5}\n",
	})
	got, err := GuestTunnels(m, h.Tunnel, h.Incus.Guests)
	if err != nil || len(got) != 2 {
		t.Fatalf("tunnels = %+v, err = %v", got, err)
	}
	lab, build := got[0], got[1]
	if lab.Link() != "wg1" || lab.Port != 51821 || lab.SlotAddress().String() != "10.99.1.1" || lab.ForwardFrom[0] != "10.99.1.2" {
		t.Errorf("lab = %+v", lab)
	}
	if build.Link() != "wg2" || build.KeyFile() != "/etc/systemd/network/60-wg2.key" || build.PeersDir() != "/etc/wireguard/wg2" || build.Table() != 1002 || len(build.Peers) != 2 {
		t.Errorf("build = %+v", build)
	}
}

func TestATunnelClashIsRefused(t *testing.T) {
	h := hostGuests(t)
	for name, tc := range map[string]struct {
		lab, build, want string
	}{
		"one slot":             {"WS_TUNNEL_ADDRESS: 10.99.1.1/24\nWS_TUNNEL_PORT: 51821\n", "WS_TUNNEL_ADDRESS: 10.99.1.1/24\nWS_TUNNEL_PORT: 51822\n", "build holds slot 1, which lab holds"},
		"one port":             {"WS_TUNNEL_ADDRESS: 10.99.1.1/24\nWS_TUNNEL_PORT: 51821\n", "WS_TUNNEL_ADDRESS: 10.99.2.1/24\nWS_TUNNEL_PORT: 51821\n", "build holds UDP port 51821, which lab holds"},
		"the host's port":      {"WS_TUNNEL_ADDRESS: 10.99.1.1/24\n", "WS_TUNNEL_ADDRESS: 10.99.2.1/24\nWS_TUNNEL_PORT: 51822\n", "lab holds UDP port 51820, which the host's own tunnel holds"},
		"no tunnel set":        {"", "WS_TUNNEL_ADDRESS: 10.99.2.1/24\nWS_TUNNEL_PORT: 51822\n", "slot 0"},
		"a slot beyond a mac":  {"WS_TUNNEL_ADDRESS: 192.168.7.1/24\nWS_TUNNEL_PEERS: []\nWS_TUNNEL_PORT: 51821\n", "WS_TUNNEL_ADDRESS: 10.99.2.1/24\nWS_TUNNEL_PORT: 51822\n", "not a slot's 10.99.N.1/24"},
		"an address off a .1":  {"WS_TUNNEL_ADDRESS: 10.99.1.5/24\nWS_TUNNEL_PEERS: []\nWS_TUNNEL_PORT: 51821\n", "WS_TUNNEL_ADDRESS: 10.99.2.1/24\nWS_TUNNEL_PORT: 51822\n", "not a slot's 10.99.N.1/24"},
		"a bad file of a peer": {"WS_TUNNEL_PORT: x\n", "", "values.yml"},
	} {
		t.Run(name, func(t *testing.T) {
			m := guestMachine(map[string]string{"lab": tc.lab, "build": tc.build})
			if _, err := GuestTunnels(m, h.Tunnel, h.Incus.Guests); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want it to say %q", err, tc.want)
			}
		})
	}
}

func TestPeersOfOneNameInTwoTunnelsAreNoClash(t *testing.T) {
	h := hostGuests(t)
	m := guestMachine(map[string]string{
		"lab":   "WS_TUNNEL_ADDRESS: 10.99.1.1/24\nWS_TUNNEL_PORT: 51821\n",
		"build": "WS_TUNNEL_ADDRESS: 10.99.2.1/24\nWS_TUNNEL_PORT: 51822\n",
	})
	got, err := GuestTunnels(m, h.Tunnel, h.Incus.Guests)
	if err != nil || got[0].Peers[0].Name != "mac" || got[1].Peers[0].Name != "mac" {
		t.Errorf("tunnels = %+v, err = %v, want a mac on each", got, err)
	}
}
