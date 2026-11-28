package values

import "fmt"

// ikewardSet reports whether c holds the ikeward keys. ikewardPairs has checked that they come
// together, so any one stands for all three.
func (c Conf) ikewardSet() bool { return c.IkewardDNS != "" }

// ikewardClashes refuses a second VPN. It refuses a domain that one file takes by ikeward and the
// other's WS_RESOLVERS names without ikeward's resolver.
func ikewardClashes(c Conf, o Sibling) []Error {
	var errs []Error
	if c.ikewardSet() && o.Conf.ikewardSet() {
		errs = append(errs, Error{Msg: fmt.Sprintf(
			"WS_IKEWARD_DNS names a VPN, as server %s's does: ikeward serves one server's profile", o.Name)})
	}
	if c.ikewardSet() {
		for _, d := range unanswered(c.IkewardDomains, c.IkewardDNS, o.Conf.Resolvers) {
			errs = append(errs, Error{Msg: fmt.Sprintf("WS_IKEWARD_DOMAINS's %s is in server %s's WS_RESOLVERS, "+
				"which lacks WS_IKEWARD_DNS=%s: /etc/resolver holds one file a domain", d, o.Name, c.IkewardDNS)})
		}
	}
	if o.Conf.ikewardSet() {
		for _, d := range unanswered(o.Conf.IkewardDomains, o.Conf.IkewardDNS, c.Resolvers) {
			errs = append(errs, Error{Msg: fmt.Sprintf("WS_RESOLVERS's %s is in server %s's WS_IKEWARD_DOMAINS, "+
				"whose WS_IKEWARD_DNS=%s it lacks: /etc/resolver holds one file a domain", d, o.Name, o.Conf.IkewardDNS)})
		}
	}
	return errs
}
