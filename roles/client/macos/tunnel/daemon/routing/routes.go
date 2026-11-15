package routing

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	tunnel "github.com/zadykian/ws/roles/client/macos/tunnel/plan"
)

// RunRoutes carries out a plan of routes into the utun tun. It returns the networks it could not
// add, in the plan's order, with a log line for each route it could not delete or left alone. It
// deletes first, as an old route may be in the way of a new one. It deletes a route that goes
// through tun alone: another server's daemon may route the network through its own utun.
func RunRoutes(ctx context.Context, m effect.Machine, plan tunnel.RoutePlan, tun string) (failed []netip.Prefix, logs []string) {
	for _, n := range plan.Delete {
		out, err := m.Run(ctx, "route", "-n", "get", "-inet", n.String())
		if err != nil {
			continue // no route to delete
		}
		route, via := routeOf(string(out))
		if route != n {
			// route get answers with the best match, as a wider route or the default, where the
			// network's own is gone. That route is not the network's to delete.
			continue
		}
		if via != tun {
			logs = append(logs, fmt.Sprintf("leaves the route to %s alone: it goes through %s, not %s", n, via, tun))
			continue
		}
		if _, err := m.Run(ctx, "route", "-q", "-n", "delete", "-inet", n.String()); err != nil {
			logs = append(logs, fmt.Sprintf("cannot remove the route to %s", n))
		}
	}
	for _, n := range plan.Add {
		if _, err := m.Run(ctx, "route", "-q", "-n", "add", "-inet", n.String(), "-interface", tun); err != nil {
			failed = append(failed, n)
		}
	}
	return failed, logs
}

// routeOf returns the network of the route that route -n get names, from its destination and mask,
// and its interface. A host route's answer may hold no mask line, so a destination with none is a
// /32. The network is invalid where the answer gives none, as the default's "default" is none.
func routeOf(out string) (route netip.Prefix, iface string) {
	var dest, mask netip.Addr
	masked := false
	for _, line := range strings.Split(out, "\n") {
		key, v, _ := strings.Cut(strings.TrimSpace(line), ":")
		v = strings.TrimSpace(v)
		switch key {
		case "destination":
			dest, _ = netip.ParseAddr(v)
		case "mask":
			mask, _ = netip.ParseAddr(v)
			masked = true
		case "interface":
			iface = v
		}
	}
	bits := 32
	if masked {
		ones, size := net.IPMask(mask.AsSlice()).Size()
		if size == 0 {
			return netip.Prefix{}, iface // a mask that is no address, or not of leading ones
		}
		bits = ones
	}
	if !dest.Is4() {
		return netip.Prefix{}, iface
	}
	return netip.PrefixFrom(dest, bits), iface
}
