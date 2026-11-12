package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/server/values"
)

// routeCheck stops the run on a route that Docker's networks would overlap, such as one a VPN keeps
// in a table of its own. It runs before daemon.json takes the pool. The routes are those of the
// moment, and a VPN that is down has none.
type routeCheck struct {
	m effect.Machine
	d values.Docker
}

// Plan reads the routes of every table, and plans no change.
func (c routeCheck) Plan(ctx context.Context) ([]host.Change, error) {
	out, err := c.m.Run(ctx, "ip", "-j", "-4", "route", "show", "table", "all")
	if err != nil {
		return nil, fmt.Errorf("reading the routes: %w", err)
	}
	routes, err := ParseRoutes(out)
	if err != nil {
		return nil, err
	}
	found, err := Conflicts(routes, c.d.AddressPool, bridgeCIDR(c.d))
	if err != nil {
		return nil, err
	}
	if len(found) > 0 {
		return nil, fmt.Errorf("the Docker networks would overlap these routes, which it does not avoid: %s. "+
			"Set docker_address_pool and docker_bridge_ip clear of them. A moved pool serves the networks created "+
			"afterwards, as by docker compose down and up. docker0 takes a moved bridge as Docker starts with no "+
			"container running, as after a reboot", strings.Join(found, ", "))
	}
	return nil, nil
}
