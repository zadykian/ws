package routing

import (
	"context"
	"fmt"
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
		if via := routeInterface(string(out)); via != tun {
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

// routeInterface returns the interface that route -n get names, empty where it names none.
func routeInterface(out string) string {
	for _, line := range strings.Split(out, "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "interface:"); ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
