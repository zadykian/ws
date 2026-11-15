package conf

import (
	"fmt"
	"net/netip"
	"strings"
)

// clashes returns a refusal for each network of c that overlaps one of another server's, each
// WS_RESOLVERS domain that another names, and a cable that another names too. It refuses c's slot
// where another holds it, and a port that another takes behind the same router. Both servers refuse
// until one changes, so no order of servers decides which does.
func clashes(c Conf, others []Sibling) []Error {
	var errs []Error
	for _, o := range others {
		errs = append(errs, networkClashes(c, o)...)
		errs = append(errs, domainClashes(c, o)...)
		errs = append(errs, slotClashes(c, o)...)
		errs = append(errs, portClashes(c, o)...)
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

// slotClashes returns a refusal where c holds o's slot: each server's tunnel is the subnet of its
// own.
func slotClashes(c Conf, o Sibling) []Error {
	if c.TunnelSlot() != o.Conf.TunnelSlot() {
		return nil
	}
	return []Error{{Msg: fmt.Sprintf("WS_SLOT=%d is server %s's slot too: each server takes its own",
		c.TunnelSlot(), o.Name)}}
}

// portClashes returns a refusal for each of c's router ports that o takes too, behind the same
// WS_REMOTE, whatever its case: a name on the internet has none.
func portClashes(c Conf, o Sibling) []Error {
	if c.Remote == "" || !strings.EqualFold(c.Remote, o.Conf.Remote) {
		return nil
	}
	var errs []Error
	for _, p := range []struct{ key, own, theirs string }{
		{"WS_PORT", c.TunnelPort(), o.Conf.TunnelPort()},
		{"WS_REMOTE_PORT", c.ForwardPort(), o.Conf.ForwardPort()},
	} {
		if p.own == p.theirs {
			errs = append(errs, Error{Msg: fmt.Sprintf(
				"%s=%s is server %s's too, behind WS_REMOTE %s: the router forwards a port to one server",
				p.key, p.own, o.Name, c.Remote)})
		}
	}
	return errs
}
