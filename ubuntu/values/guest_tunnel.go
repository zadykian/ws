package values

import (
	"errors"
	"fmt"
	"net/netip"
	"path"
	"strconv"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/internal/slot"
)

// GuestTunnel is the tunnel that the host ends for a guest. Its slot, port and peers are the
// guest's own values, as the Mac names them for a server that ends its own tunnel.
type GuestTunnel struct {
	Guest Guest
	Slot  slot.Slot
	Port  int
	Peers []TunnelPeer
	// ForwardFrom are the peers that the guest's ikeward forwards: the ACL lets them reach any
	// destination, as ikeward keeps no rule of its own.
	ForwardFrom []string
}

// Link is the name of the guest's WireGuard interface on the host. It names the slot, not the
// guest, as an interface's name holds 15 bytes at most. wg0 is the host's own.
func (t GuestTunnel) Link() string { return "wg" + strconv.Itoa(int(t.Slot)) }

// SlotAddress is the slot's server end, 10.99.N.1, which the guest holds and the host routes to it.
func (t GuestTunnel) SlotAddress() netip.Addr { return t.Slot.Server() }

// KeyFile is the interface's private key, which networkd reads.
func (t GuestTunnel) KeyFile() string { return "/etc/systemd/network/60-" + t.Link() + ".key" }

// PeersDir holds a public key a peer, as NAME.pub. A directory a slot keeps each tunnel's peers
// apart, as every tunnel of a Mac names its peer mac.
func (t GuestTunnel) PeersDir() string { return path.Join("/etc/wireguard", t.Link()) }

// Table is the routing table whose one route sends a peer's packets to the guest, and the priority of
// the rule that selects it. Both name the slot, and sit clear of the tables of WireGuard's tools.
func (t GuestTunnel) Table() int { return 1000 + int(t.Slot) }

// Tunnel returns the tunnel that the host ends for the guest, from the guest's own values. The
// address must be a slot's 10.99.N.1/24, N from 1 to slot.Last. The host routes it to the guest,
// and slot 0 is wg0's, the host's own. A guest that sets no tunnel holds the defaults, which the
// host's own tunnel holds too, so the clash refuses it.
func (g Guest) Tunnel(own Values) (GuestTunnel, error) {
	p, err := netip.ParsePrefix(own.Tunnel.Address)
	if err != nil {
		return GuestTunnel{}, fmt.Errorf("%s: WS_TUNNEL_ADDRESS %q is no address with its prefix length", g.Name, own.Tunnel.Address)
	}
	s, ok := slot.Find(p.Addr())
	switch {
	case !ok || p != s.Wg0():
		return GuestTunnel{}, fmt.Errorf("%s: WS_TUNNEL_ADDRESS is %s, not a slot's 10.99.N.1/24, N from 1 to %d: the host routes the slot to the guest", g.Name, p, slot.Last)
	case s == 0:
		return GuestTunnel{}, fmt.Errorf("%s: WS_TUNNEL_ADDRESS is %s, slot 0, which is wg0's: the host's own tunnel keeps it. Set a slot from 1 to %d in %s", g.Name, p, slot.Last, g.File())
	}
	return GuestTunnel{Guest: g, Slot: s, Port: own.Tunnel.Port, Peers: own.Tunnel.Peers, ForwardFrom: own.Ikeward.ForwardFrom}, nil
}

// GuestTunnels reads each guest's own values on m and returns the tunnels that the host ends, in the
// guests' order. It refuses a slot or a port that two tunnels share, the host's own among them: two
// tunnels with one slot want one interface, one address and one port. Peers of one name in two
// tunnels are no clash, as each tunnel has its own directory of keys.
func GuestTunnels(m effect.Machine, own Tunnel, guests []Guest) ([]GuestTunnel, error) {
	var tunnels []GuestTunnel
	for _, g := range guests {
		_, v, err := g.Read(m)
		if err != nil {
			return nil, err
		}
		t, err := g.Tunnel(v)
		if err != nil {
			return nil, err
		}
		tunnels = append(tunnels, t)
	}
	return tunnels, tunnelsCheck(own, tunnels)
}

// tunnelsCheck refuses the tunnels whose slot or port another holds, or the host's own tunnel.
func tunnelsCheck(own Tunnel, tunnels []GuestTunnel) error {
	var problems []string
	ownSlot, hasOwnSlot := slot.Slot(0), false
	if p, err := netip.ParsePrefix(own.Address); err == nil {
		ownSlot, hasOwnSlot = slot.Find(p.Addr())
	}
	slots := map[slot.Slot]string{}
	ports := map[int]string{own.Port: "the host's own tunnel"}
	if hasOwnSlot {
		slots[ownSlot] = "the host's own tunnel"
	}
	for _, t := range tunnels {
		if holder, taken := slots[t.Slot]; taken {
			problems = append(problems, fmt.Sprintf("%s holds slot %d, which %s holds too.", t.Guest.Name, t.Slot, holder))
		}
		if holder, taken := ports[t.Port]; taken {
			problems = append(problems, fmt.Sprintf("%s holds UDP port %d, which %s holds too.", t.Guest.Name, t.Port, holder))
		}
		slots[t.Slot], ports[t.Port] = t.Guest.Name, t.Guest.Name
	}
	if len(problems) == 0 {
		return nil
	}
	return errors.New("the tunnels clash: " + strings.Join(problems, " ") +
		" Each guest takes a slot and a port of its own in WS_TUNNEL_ADDRESS and WS_TUNNEL_PORT.")
}
