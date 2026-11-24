package values

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"

	"github.com/zadykian/ws/internal/slot"
)

// The Mac's scripts fix wg0 at a slot's .1 and the Mac at its .2 (internal/slot). They fix
// the peer's name too, as the mac.pub that the Mac's runbook has the user copy to the server.
const (
	macPeer = "mac"
	maxPort = 65535
)

// check refuses wg0's address and port, and the peers, that the tunnel role cannot use. The address
// is wg0's: an IPv4 address of a host, with the prefix length of the tunnel's subnet. Each peer has
// a name of lowercase letters, digits and dashes, which names its key's file. It has an IPv4
// address of its own too, a host's in that subnet but not wg0's. With the peer mac, wg0 and mac
// take the addresses the Mac's scripts fix. It names every problem it finds.
func (t Tunnel) check() error {
	var problems []string
	if t.Port < 1 || t.Port > maxPort {
		problems = append(problems, fmt.Sprintf("WS_TUNNEL_PORT is %d, not a port from 1 to %d.", t.Port, maxPort))
	}
	server, err := netip.ParsePrefix(t.Address)
	if err != nil || !server.Addr().Is4() {
		problems = append(problems, fmt.Sprintf("WS_TUNNEL_ADDRESS is %s, not an IPv4 address with its prefix length, as %s.", t.Address, slot.Slot(0).Wg0()))
		return errors.New(strings.Join(problems, " "))
	}
	subnet := server.Masked()
	if !isHost(server.Addr(), subnet) {
		problems = append(problems, fmt.Sprintf("WS_TUNNEL_ADDRESS is %s, not a host's address in %s.", t.Address, subnet))
	}
	problems = append(problems, peerProblems(t, server)...)
	if len(problems) == 0 {
		return nil
	}
	return errors.New(strings.Join(problems, " "))
}

// isHost reports whether ip is a host's address in subnet: a network keeps its first and last
// addresses for itself and its broadcast where it has more than two.
func isHost(ip netip.Addr, subnet netip.Prefix) bool {
	if !subnet.Contains(ip) {
		return false
	}
	return subnet.Bits() > longestBroadcastPrefix || (ip != subnet.Addr() && ip != broadcast(subnet))
}
