package conf

import (
	"fmt"
	"net/netip"
	"strings"
)

// clashes returns a refusal for each network of c that overlaps one of another server's. Both
// servers refuse until one changes, so no order of servers decides which does.
func clashes(c Conf, others []Sibling) []Error {
	var errs []Error
	for _, o := range others {
		errs = append(errs, networkClashes(c, o)...)
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
