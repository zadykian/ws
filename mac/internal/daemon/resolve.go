package daemon

import (
	"context"
	"net/netip"
	"strings"

	"github.com/zadykian/ws/mac/internal/effect"
)

// Resolve returns the IPv4 address of name from the system's resolver, as dscacheutil gives it,
// or an invalid address where it has none. It takes the first answer, as wg would take
// getaddrinfo's, which may be IPv6, and the router forwards IPv4 alone. So an IPv6 first answer
// is no address.
func Resolve(ctx context.Context, m effect.Machine, name string) netip.Addr {
	if a, err := netip.ParseAddr(name); err == nil {
		if a.Is4() {
			return a
		}
		return netip.Addr{}
	}
	out, err := m.Run(ctx, "dscacheutil", "-q", "host", "-a", "name", name)
	if err != nil {
		return netip.Addr{}
	}
	for _, line := range strings.Split(string(out), "\n") {
		if f := strings.Fields(line); len(f) >= 2 && f[0] == "ip_address:" {
			if a, err := netip.ParseAddr(f[1]); err == nil && a.Is4() {
				return a
			}
			return netip.Addr{}
		}
	}
	return netip.Addr{}
}
