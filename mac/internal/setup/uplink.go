package setup

import (
	"net/netip"
	"strings"

	"github.com/zadykian/ws/mac/internal/check"
	"github.com/zadykian/ws/mac/internal/tunnel"
)

// uplinkRouter returns the router that info shows outside the cable's /30, which is an uplink's:
// the cable, with no DHCP, gives none. It is empty where the router is the cable's or none.
func uplinkRouter(info string) string {
	for _, line := range strings.Split(info, "\n") {
		router, ok := strings.CutPrefix(strings.TrimRight(line, " \t\r"), "Router: ")
		if !ok || router == "" || router[0] < '0' || router[0] > '9' {
			continue
		}
		a, err := netip.ParseAddr(router)
		if !check.IPv4(router) || err != nil || !tunnel.CableNet.Contains(a) {
			return router
		}
	}
	return ""
}
