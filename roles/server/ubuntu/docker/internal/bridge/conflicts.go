package bridge

import (
	"cmp"
	"fmt"
	"net/netip"
)

// shortestPrefix is the shortest prefix of a route that counts. A shorter one is taken for a VPN's
// catch-all, as 0.0.0.0/1, rather than a network.
const shortestPrefix = 8

// network returns the network that value names. ip writes a route to one address without its /32,
// and a prefix may carry host bits, as the bridge's 172.17.0.1/16 does.
func network(value string) (netip.Prefix, error) {
	if p, err := netip.ParsePrefix(value); err == nil {
		return p.Masked(), nil
	}
	a, err := netip.ParseAddr(value)
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("%q is not a network", value)
	}
	return netip.PrefixFrom(a, a.BitLen()), nil
}

// overlaps returns, as text, the routes that Docker's networks would overlap. One is any route
// into pool that is not an on-link route of the main table, which Docker avoids. The other is any
// route into bridge but docker0's own. Routes of another type than unicast do not count. Nor do
// those shorter than /8, a VPN's 0.0.0.0/1, which cover every address rather than name a network.
func overlaps(routes []route, pool, bridge string) ([]string, error) {
	poolNet, err := network(pool)
	if err != nil {
		return nil, err
	}
	bridgeNet, err := network(bridge)
	if err != nil {
		return nil, err
	}
	var found []string
	for _, r := range routes {
		if (r.Type != "" && r.Type != "unicast") || r.Dst == "" || r.Dst == "default" {
			continue
		}
		n, err := network(r.Dst)
		if err != nil {
			return nil, err
		}
		if n.Bits() < shortestPrefix {
			continue
		}
		table, dev := cmp.Or(r.Table, "main"), cmp.Or(r.Dev, "-")
		onLink := table == "main" && r.Scope == "link"
		if (n.Overlaps(bridgeNet) && dev != "docker0") || (n.Overlaps(poolNet) && !onLink) {
			found = append(found, fmt.Sprintf("%s dev %s table %s", n, dev, table))
		}
	}
	return found, nil
}
