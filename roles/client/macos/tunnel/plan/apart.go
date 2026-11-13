package plan

import (
	"fmt"
	"maps"
	"net/netip"
	"slices"
)

// Apart returns the networks of own that overlap no network of held, and a warning for each it
// leaves out. held maps a server's name to the networks its routes.d holds. Both daemons leave a
// shared network out, so neither depends on which reads first.
func Apart(own []netip.Prefix, held map[string][]netip.Prefix) (kept []netip.Prefix, warnings []string) {
	servers := slices.Sorted(maps.Keys(held))
next:
	for _, n := range own {
		for _, server := range servers {
			for _, h := range held[server] {
				if n.Overlaps(h) {
					warnings = append(warnings, fmt.Sprintf("leaves %s out: server %s's routes.d holds %s", n, server, h))
					continue next
				}
			}
		}
		kept = append(kept, n)
	}
	return kept, warnings
}
