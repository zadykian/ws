package settings

import (
	"fmt"
	"net/netip"
	"regexp"

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
			problems = append(problems, fmt.Sprintf("%s's address, %s, is not an IPv4 address, as %s.", p.Name, p.Address, macClient))
			continue
		}
		if p.Name == macPeer && ip.String() != macClient {
			problems = append(problems, fmt.Sprintf("mac's address is %s, but the Mac's scripts fix it at %s.", ip, macClient))
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
	if names[macPeer] && server.String() != macServer {
		problems = append(problems, fmt.Sprintf("wireguard_address is %s, but the Mac's scripts fix wg0 at %s.", server, macServer))
	}
	return problems
}
