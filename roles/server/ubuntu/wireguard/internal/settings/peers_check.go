package settings

import (
	"fmt"
	"net/netip"
	"regexp"

	"github.com/zadykian/ws/roles/internal/slot"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

var peerName = regexp.MustCompile(`^[a-z0-9-]+$`)

// peerProblems names what is wrong with the peers, given wg0's address with its prefix length.
func peerProblems(w values.Wireguard, server netip.Prefix) []string {
	var problems []string
	names := map[string]bool{}
	taken := map[netip.Addr]string{server.Addr(): "wg0"}
	subnet := server.Masked()
	for _, p := range w.Peers {
		switch {
		case !peerName.MatchString(p.Name):
			problems = append(problems, fmt.Sprintf("A peer's name, %q, isn't lowercase letters, digits and dashes.", p.Name))
		case names[p.Name]:
			problems = append(problems, fmt.Sprintf("Two peers are named %s.", p.Name))
		}
		names[p.Name] = true
		ip, err := netip.ParseAddr(p.Address)
		if err != nil || !ip.Is4() {
			problems = append(problems, fmt.Sprintf("%s's address, %s, is not an IPv4 address, as %s.", p.Name, p.Address, slot.Slot(0).Mac()))
			continue
		}
		switch holder, held := taken[ip]; {
		case !isHost(ip, subnet):
			problems = append(problems, fmt.Sprintf("%s's address, %s, is not a host's address in %s.", p.Name, ip, subnet))
		case held:
			problems = append(problems, fmt.Sprintf("%s's address, %s, is %s's too.", p.Name, ip, holder))
		default:
			taken[ip] = p.Name
		}
	}
	return append(problems, macProblems(w, server)...)
}

// macProblems checks the peer mac against the slot that wg0's address names: wg0 at that slot's
// .1 and the Mac at its .2. Without the peer, wg0 may sit anywhere.
func macProblems(w values.Wireguard, server netip.Prefix) []string {
	var mac *values.WireguardPeer
	for i := range w.Peers {
		if w.Peers[i].Name == macPeer {
			mac = &w.Peers[i]
		}
	}
	if mac == nil {
		return nil
	}
	s, ok := slot.Find(server.Addr())
	if !ok || server != s.Wg0() {
		return []string{fmt.Sprintf("wireguard_address is %s, but the Mac's scripts fix wg0 at a slot's 10.99.N.1/24, N from 0 to %d.", server, slot.Last)}
	}
	if ip, err := netip.ParseAddr(mac.Address); err == nil && ip.Is4() && ip != s.Mac() {
		return []string{fmt.Sprintf("mac's address is %s, but the Mac's scripts fix it at %s.", ip, s.Mac())}
	}
	return nil
}
