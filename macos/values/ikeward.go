package values

import (
	"fmt"
	"net/netip"
	"slices"
	"strings"
)

// ikewardPairs checks that the three ikeward keys come together and the resolver lies in a floor.
// It checks that a domain that WS_RESOLVERS names too names the resolver there.
func ikewardPairs(c Conf) []Error {
	held := 0
	for _, v := range []string{c.IkewardDNS, c.IkewardDomains, c.IkewardFloors} {
		if v != "" {
			held++
		}
	}
	switch {
	case held == 0:
		return nil
	case held != 3:
		return []Error{{0, "WS_IKEWARD_DNS, WS_IKEWARD_DOMAINS and WS_IKEWARD_FLOORS go together"}}
	case !inFloor(c.IkewardDNS, c.IkewardFloors):
		return []Error{{0, "WS_IKEWARD_DNS=" + c.IkewardDNS + " is in no floor of WS_IKEWARD_FLOORS: " +
			"ikeward routes a resolver inside a floor alone"}}
	}
	if apart := unanswered(c.IkewardDomains, c.IkewardDNS, c.Resolvers); len(apart) > 0 {
		return []Error{{0, fmt.Sprintf("WS_IKEWARD_DOMAINS's %s is a WS_RESOLVERS domain too, which lacks "+
			"WS_IKEWARD_DNS=%s: /etc/resolver holds one file a domain", apart[0], c.IkewardDNS)}}
	}
	return nil
}

// inFloor reports whether the address dns lies in one of the networks of floors. Both passed their
// own checks.
func inFloor(dns, floors string) bool {
	a := netip.MustParseAddr(dns)
	return slices.ContainsFunc(strings.Fields(floors), func(f string) bool {
		return netip.MustParsePrefix(f).Contains(a)
	})
}

// unanswered returns the domains that the resolvers list names without the address dns, in order.
func unanswered(domains, dns, resolvers string) []string {
	answers := map[string][]string{}
	for _, pair := range strings.Fields(resolvers) {
		domain, addresses, _ := strings.Cut(pair, "=")
		answers[domain] = strings.Split(addresses, ",")
	}
	var apart []string
	for _, d := range strings.Fields(domains) {
		if addresses, named := answers[d]; named && !slices.Contains(addresses, dns) {
			apart = append(apart, d)
		}
	}
	return apart
}
