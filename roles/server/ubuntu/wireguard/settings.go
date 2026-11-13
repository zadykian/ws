// Package wireguard is the wireguard role as Go steps. It sets up wg0, the server's end of a
// WireGuard tunnel for the Mac, from networkd's own files. It also sets up the drops that keep
// Docker's containers off the Mac. See docs/wireguard.md.
package wireguard

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"

	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// The Mac's daemon, in roles/client/macos/tunnel/plan, fixes the tunnel's ends, the server and the address. It
// also fixes the peer's name, as the mac.pub that `ws client macos apply` has the user write on the
// server.
const (
	macPeer   = "mac"
	macServer = "10.99.0.1/24"
	macClient = "10.99.0.2"
	maxPort   = 65535
	// longestBroadcastPrefix is the longest prefix of a subnet of more than two addresses, which
	// keeps its first and its last for itself and its broadcast. A /31 has neither (RFC 3021), nor
	// has a /32, a single address.
	longestBroadcastPrefix = 30
)

// Settings refuses wg0's address and port, and the peers, that the role cannot use. The address is
// wg0's: an IPv4 address of a host, with the prefix length of the tunnel's subnet. Each peer has a
// name of lowercase letters, digits and dashes, which names its key's file. It has an IPv4 address
// of its own too, a host's in that subnet but not wg0's. With the peer mac, wg0 and mac take the
// addresses the Mac's scripts fix. It names every problem it finds.
func Settings(w values.Wireguard) error {
	var problems []string
	if w.Port < 1 || w.Port > maxPort {
		problems = append(problems, fmt.Sprintf("wireguard_port is %d, not a port from 1 to %d.", w.Port, maxPort))
	}
	server, err := netip.ParsePrefix(w.Address)
	if err != nil || !server.Addr().Is4() {
		problems = append(problems, fmt.Sprintf("wireguard_address is %s, not an IPv4 address with its prefix length, as %s.", w.Address, macServer))
		return errors.New(strings.Join(problems, " "))
	}
	subnet := server.Masked()
	if !isHost(server.Addr(), subnet) {
		problems = append(problems, fmt.Sprintf("wireguard_address is %s, not a host's address in %s.", w.Address, subnet))
	}
	problems = append(problems, peerProblems(w, server)...)
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
	return subnet.Bits() > longestBroadcastPrefix || (ip != subnet.Addr() && ip != last(subnet))
}

// last returns the last address of an IPv4 subnet.
func last(subnet netip.Prefix) netip.Addr {
	b := subnet.Addr().As4()
	for i := range b {
		bits := min(max(subnet.Bits()-8*i, 0), 8)
		b[i] |= byte(0xff >> bits)
	}
	return netip.AddrFrom4(b)
}
