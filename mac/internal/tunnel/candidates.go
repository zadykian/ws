package tunnel

import (
	"fmt"
	"net/netip"
)

// Tier is a kind of path to ws.
type Tier string

// The tiers, in the order the daemon tries them.
const (
	TierCable  Tier = "cable"
	TierLAN    Tier = "lan"
	TierRemote Tier = "remote"
)

// Iface is one IPv4 address of the Mac, on an interface that is up, the tunnel's aside.
type Iface struct {
	Name string
	Addr netip.Addr
}

// Candidate is an endpoint to try, with the tier it belongs to.
type Candidate struct {
	Tier Tier
	Host netip.Addr
}

// Candidates lists the endpoints to try, in order: the cable, where the Mac has an address in its
// subnet; the LAN, where it has one in WS_LAN; then WS_REMOTE, as resolved. resolved is invalid
// where the name has no IPv4 address, and then why says so. The caller resolves the name anew
// before each choice, as an address may change.
func Candidates(ifaces []Iface, c Conf, resolved netip.Addr) (found []Candidate, why []string) {
	within := func(n netip.Prefix) bool {
		for _, each := range ifaces {
			if n.Contains(each.Addr) {
				return true
			}
		}
		return false
	}
	if within(CableNet) {
		found = append(found, Candidate{TierCable, CableServer})
	}
	if c.LAN.IsValid() && c.LANServer.IsValid() && within(c.LAN) {
		found = append(found, Candidate{TierLAN, c.LANServer})
	}
	if c.Remote != "" {
		if resolved.IsValid() {
			found = append(found, Candidate{TierRemote, resolved})
		} else {
			why = append(why, fmt.Sprintf("%s has no IPv4 address", c.Remote))
		}
	}
	return found, why
}
