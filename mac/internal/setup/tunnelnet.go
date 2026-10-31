package setup

import "strings"

// routesText is routes.d/mac-conf: the networks of WS_TUNNEL_NETWORKS, a network a line, which the
// daemon adds to the peer's AllowedIPs and routes into its utun.
func routesText(networks []string) string {
	return "# Written by ws's mac/lib.sh: ws_tunnel_routes mac-conf.\n" + strings.Join(networks, "\n") + "\n"
}

// resolverHeader is the first line of the files in /etc/resolver that ws writes, by which a run
// knows them for its own.
const resolverHeader = "# Written by ws's mac/setup.sh from WS_RESOLVERS in ~/.config/ws/mac.conf."

// resolverText is /etc/resolver/DOMAIN: a nameserver line for each address (resolver(5)).
func resolverText(addresses []string) string {
	return resolverHeader + "\n" + "nameserver " + strings.Join(addresses, "\nnameserver ") + "\n"
}

// resolver is one pair of WS_RESOLVERS, DOMAIN=ADDRESS[,ADDRESS...].
type resolver struct {
	Domain    string
	Addresses []string
}

// resolvers splits WS_RESOLVERS, which conf checked.
func resolvers(list string) []resolver {
	var rs []resolver
	for _, pair := range strings.Fields(list) {
		domain, addresses, _ := strings.Cut(pair, "=")
		rs = append(rs, resolver{domain, strings.Split(addresses, ",")})
	}
	return rs
}
