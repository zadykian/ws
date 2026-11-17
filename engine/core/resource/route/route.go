// Package route checks the routes, and an interface's address, against the networks that a
// program takes, before it takes them.
package route

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
)

// shortestPrefix is the shortest prefix of a route that counts. A shorter one is taken for a VPN's
// catch-all, as 0.0.0.0/1, rather than a network.
const shortestPrefix = 8

// Clear stops the run on a route that a program's networks would overlap, such as one a VPN keeps
// in a table of its own. Pool is the range that the program takes its networks from. A route into
// it counts, but for an on-link route of the main table, which the program avoids. Net is a
// network that Dev alone routes: a route into it on another device counts. Hint says how to clear
// an overlap. The routes are those of the moment, and a VPN that is down has none.
type Clear struct {
	M         effect.Machine
	Pool, Net netip.Prefix
	Dev       string
	Hint      string
}

// entry is one route of `ip -j -4 route show table all`.
type entry struct {
	Dst   string `json:"dst"`
	Type  string `json:"type"`
	Table string `json:"table"`
	Dev   string `json:"dev"`
	Scope string `json:"scope"`
}

// Plan reads the routes of every table, and plans no change. The error names each route that
// overlaps, and ends with Hint.
func (c Clear) Plan(ctx context.Context) ([]resource.Change, error) {
	out, err := c.M.Run(ctx, "ip", "-j", "-4", "route", "show", "table", "all")
	if err != nil {
		return nil, fmt.Errorf("read the routes: %w", err)
	}
	var routes []entry
	if err := json.Unmarshal(out, &routes); err != nil {
		return nil, fmt.Errorf("read the routes: %w", err)
	}
	found, err := c.overlaps(routes)
	if err != nil || len(found) == 0 {
		return nil, err
	}
	return nil, fmt.Errorf("%s and %s would overlap these routes: %s. %s", c.Pool, c.Net, strings.Join(found, ", "), c.Hint)
}

// overlaps returns, as text, the routes that Pool or Net would overlap. Routes of another type than
// unicast do not count, nor do those shorter than shortestPrefix.
func (c Clear) overlaps(routes []entry) ([]string, error) {
	pool, own := c.Pool.Masked(), c.Net.Masked()
	var found []string
	for _, r := range routes {
		if (r.Type != "" && r.Type != "unicast") || r.Dst == "" || r.Dst == "default" {
			continue
		}
		n, err := network(r.Dst)
		if err != nil {
			return nil, err
		}
		if n.Bits() < shortestPrefix {
			continue
		}
		table, dev := cmp.Or(r.Table, "main"), cmp.Or(r.Dev, "-")
		onLink := table == "main" && r.Scope == "link"
		if (n.Overlaps(own) && dev != c.Dev) || (n.Overlaps(pool) && !onLink) {
			found = append(found, fmt.Sprintf("%s dev %s table %s", n, dev, table))
		}
	}
	return found, nil
}

// network returns the network of a route's destination. ip writes a route to one address without
// its /32.
func network(dst string) (netip.Prefix, error) {
	if p, err := netip.ParsePrefix(dst); err == nil {
		return p.Masked(), nil
	}
	a, err := netip.ParseAddr(dst)
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("the route to %q names no network", dst)
	}
	return netip.PrefixFrom(a, a.BitLen()), nil
}
