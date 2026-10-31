package daemon

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/tunnel"
)

// RunRoutes carries out a plan of routes into the utun tun, and returns the networks it could not
// add, in the plan's order, with a log line for each route it could not delete. It deletes first,
// as an old route may be in the way of a new one.
func RunRoutes(ctx context.Context, m effect.Machine, plan tunnel.RoutePlan, tun string) (failed []netip.Prefix, logs []string) {
	for _, n := range plan.Delete {
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
