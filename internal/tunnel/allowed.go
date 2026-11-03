package tunnel

import (
	"fmt"
	"net/netip"
	"slices"
)

// Allowed is what the peer takes with some endpoints.
type Allowed struct {
	Want []netip.Prefix // the networks of routes.d that the tunnel takes
	Out  []netip.Prefix // those it leaves out, as each holds an endpoint
	Logs []string
}

// AllowedFor sorts the networks of routes.d by the endpoints. A network that holds an endpoint
// would send the tunnel's own packets into the tunnel, so it goes out. told holds the networks
// already logged as out, so that a network is logged once until it goes back in.
func AllowedFor(routes []netip.Prefix, ends []netip.Addr, told []netip.Prefix) Allowed {
	var a Allowed
	for _, n := range routes {
		i := slices.IndexFunc(ends, n.Contains)
		if i < 0 {
			a.Want = append(a.Want, n)
			continue
		}
		if !slices.Contains(told, n) {
			a.Logs = append(a.Logs,
				fmt.Sprintf("leaves %s out of the tunnel while it holds the endpoint, %s", n, ends[i]))
		}
		a.Out = append(a.Out, n)
	}
	return a
}

// IPs is the peer's AllowedIPs: ws's address, then the networks wanted.
func (a Allowed) IPs() []netip.Prefix {
	return append([]netip.Prefix{netip.PrefixFrom(Server, 32)}, a.Want...)
}
