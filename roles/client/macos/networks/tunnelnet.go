package networks

import "strings"

// routesText is a file of a server's routes.d: the networks of a set, a network a line. The
// server's daemon adds them to the peer's AllowedIPs and routes them into its utun.
func routesText(server, set string, networks []string) string {
	return "# Written by github.com/zadykian/ws: ws client macos tunnel routes --server " + server + " " + set + ".\n" +
		strings.Join(networks, "\n") + "\n"
}

// resolverHeader is the first line of the files in /etc/resolver that ws writes, by which a run
// knows them for its own.
const resolverHeader = "# Written by github.com/zadykian/ws: ws client macos apply, from WS_RESOLVERS in ~/.config/ws/mac.conf."

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
