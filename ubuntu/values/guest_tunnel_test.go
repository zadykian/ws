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

// loadedHost loads the host's file with each guest's own file, as the host's commands do.
func loadedHost(t *testing.T, hostFile string, files map[string]string) Host {
	m := guestMachine(files)
	m.Files["/etc/ws/values.yml"] = []byte(hostFile)
	h, err := LoadHost(m, "/etc/ws/values.yml")
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
	h := loadedHost(t, labAndBuild, map[string]string{
		"lab":   "WS_TUNNEL_ADDRESS: 10.99.1.1/24\nWS_TUNNEL_PORT: 51821\nWS_IKEWARD_FORWARD_FROM: [10.99.1.2]\n",
		"build": "WS_TUNNEL_ADDRESS: 10.99.2.1/24\nWS_TUNNEL_PORT: 51822\nWS_TUNNEL_PEERS:\n  - {name: mac, address: 10.99.2.2}\n  - {name: tv, address: 10.99.2.5}\n",
	})
	got, err := h.GuestTunnels()
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
			h := loadedHost(t, labAndBuild, map[string]string{"lab": tc.lab, "build": tc.build})
			if _, err := h.GuestTunnels(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want it to say %q", err, tc.want)
			}
		})
	}
}

// The host's own tunnel may take any subnet where it has no peer mac. Each slot inside that subnet
// is the host's, whatever wg0's own address: lab's slot 3 is in each, build's slot 12 in the /20
// alone.
func TestAGuestsSlotInsideTheHostsOwnSubnetIsRefused(t *testing.T) {
	for address, want := range map[string][]string{
		"10.99.3.1/24": {"lab holds slot 3, 10.99.3.0/24, inside the host's own tunnel's 10.99.3.0/24."},
		"10.99.3.5/24": {"lab holds slot 3, 10.99.3.0/24, inside the host's own tunnel's 10.99.3.0/24."},
		"10.99.0.1/20": {"lab holds slot 3, 10.99.3.0/24, inside the host's own tunnel's 10.99.0.0/20.",
			"build holds slot 12, 10.99.12.0/24, inside the host's own tunnel's 10.99.0.0/20."},
	} {
		h := loadedHost(t, labAndBuild+"WS_TUNNEL_ADDRESS: "+address+"\nWS_TUNNEL_PORT: 51830\nWS_TUNNEL_PEERS: []\n", map[string]string{
			"lab":   "WS_TUNNEL_ADDRESS: 10.99.3.1/24\nWS_TUNNEL_PORT: 51821\n",
			"build": "WS_TUNNEL_ADDRESS: 10.99.12.1/24\nWS_TUNNEL_PORT: 51822\n",
		})
		_, err := h.GuestTunnels()
		if err == nil || strings.Count(err.Error(), " holds ") != len(want) {
			t.Errorf("host %s: err = %v, want %q", address, err, want)
			continue
		}
		for _, w := range want {
			if !strings.Contains(err.Error(), w) {
				t.Errorf("host %s: err = %v, want it to say %q", address, err, w)
			}
		}
	}
}

func TestPeersOfOneNameInTwoTunnelsAreNoClash(t *testing.T) {
	h := loadedHost(t, labAndBuild, map[string]string{
		"lab":   "WS_TUNNEL_ADDRESS: 10.99.1.1/24\nWS_TUNNEL_PORT: 51821\n",
		"build": "WS_TUNNEL_ADDRESS: 10.99.2.1/24\nWS_TUNNEL_PORT: 51822\n",
	})
	got, err := h.GuestTunnels()
	if err != nil || got[0].Peers[0].Name != "mac" || got[1].Peers[0].Name != "mac" {
		t.Errorf("tunnels = %+v, err = %v, want a mac on each", got, err)
	}
}

// The load holds a bad guest file as the guest's reason, so that the roles that need no tunnel run
// with it. The strict list names the reason, and a clash is no load's business.
func TestTheLoadKeepsAGuestWithABadFileAndTheListNamesIt(t *testing.T) {
	h := loadedHost(t, labAndBuild, map[string]string{
		"lab":   "WS_TUNNEL_PORT: x\n",
		"build": "WS_TUNNEL_ADDRESS: 10.99.2.1/24\nWS_TUNNEL_PORT: 51822\n",
	})
	if len(h.Incus.Guests) != 2 {
		t.Fatalf("guests = %+v, want both", h.Incus.Guests)
	}
	if _, err := h.Incus.Guests[0].Resolved(); err == nil || !strings.Contains(err.Error(), h.Incus.Guests[0].File()) {
		t.Errorf("lab's reason = %v, want it to name its file", err)
	}
	if got := h.Incus.Guests[1].Link(); got != "wg2" {
		t.Errorf("build's link = %q, want its own tunnel kept", got)
	}
	if _, err := h.GuestTunnels(); err == nil || !strings.Contains(err.Error(), h.Incus.Guests[0].File()) {
		t.Errorf("GuestTunnels = %v, want lab's reason", err)
	}
}

func TestTheLoadHoldsAClashAndTheListRefusesIt(t *testing.T) {
	h := loadedHost(t, labAndBuild, map[string]string{
		"lab":   "WS_TUNNEL_ADDRESS: 10.99.1.1/24\nWS_TUNNEL_PORT: 51821\n",
		"build": "WS_TUNNEL_ADDRESS: 10.99.1.1/24\nWS_TUNNEL_PORT: 51822\n",
	})
	if _, err := h.GuestTunnels(); err == nil || !strings.Contains(err.Error(), "the tunnels clash") {
		t.Errorf("GuestTunnels = %v, want the clash", err)
	}
	if h.Incus.Guests[0].Link() != "wg1" || h.Incus.Guests[1].Link() != "wg1" {
		t.Error("the load dropped a tunnel of a clash")
	}
}

func TestAGuestResolvesInMemoryAndAnUnloadedOneHasNoTunnel(t *testing.T) {
	own := Default()
	own.Tunnel.Address = "10.99.4.1/24"
	g := Guest{Name: "lab", Address: "10.100.0.10"}.Resolve(own)
	if tun, err := g.Resolved(); err != nil || tun.Slot != 4 || g.Mark() != 0x1004 {
		t.Errorf("resolved = %+v, %v, mark %#x", tun, err, g.Mark())
	}
	if _, err := (Guest{Name: "lab"}).Resolved(); err == nil || !strings.Contains(err.Error(), "not loaded") {
		t.Errorf("an unloaded guest: err = %v", err)
	}
}
