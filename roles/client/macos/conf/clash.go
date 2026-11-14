package conf

import (
	"fmt"
	"net/netip"
	"strings"
)

// clashes returns a refusal for each network of c that overlaps one of another server's, each
// WS_RESOLVERS domain that another names, and a cable that another names too. Both servers refuse
// until one changes, so no order of servers decides which does.
func clashes(c Conf, others []Sibling) []Error {
	var errs []Error
	for _, o := range others {
		errs = append(errs, networkClashes(c, o)...)
		errs = append(errs, domainClashes(c, o)...)
		if c.CableService != "" && o.Conf.CableService != "" {
			errs = append(errs, Error{Msg: fmt.Sprintf(
				"WS_CABLE_SERVICE names a cable, as server %s's does: the cable reaches one server", o.Name)})
		}
	}
	return errs
}

// networkClashes returns a refusal for each network of c that overlaps one of o's.
func networkClashes(c Conf, o Sibling) []Error {
	var errs []Error
	for _, own := range strings.Fields(c.TunnelNetworks) {
		a, err := netip.ParsePrefix(own)
		if err != nil {
			continue
		}
		for _, theirs := range strings.Fields(o.Conf.TunnelNetworks) {
			b, err := netip.ParsePrefix(theirs)
			if err != nil {
				continue
			}
			if !a.Contains(b.Masked().Addr()) && !b.Contains(a.Masked().Addr()) {
				continue
			}
			errs = append(errs, Error{Msg: fmt.Sprintf(
				"WS_TUNNEL_NETWORKS's %s overlaps server %s's %s: a network goes through one server's tunnel",
				own, o.Name, theirs)})
		}
	}
	return errs
}

// domainClashes returns a refusal for each domain of c's WS_RESOLVERS that o's names too.
func domainClashes(c Conf, o Sibling) []Error {
	theirs := map[string]bool{}
	for _, pair := range strings.Fields(o.Conf.Resolvers) {
		domain, _, _ := strings.Cut(pair, "=")
		theirs[domain] = true
	}
	var errs []Error
	for _, pair := range strings.Fields(c.Resolvers) {
		domain, _, _ := strings.Cut(pair, "=")
		if theirs[domain] {
			errs = append(errs, Error{Msg: fmt.Sprintf(
				"WS_RESOLVERS's %s is server %s's too: /etc/resolver holds one file a domain", domain, o.Name)})
		}
	}
	return errs
}
