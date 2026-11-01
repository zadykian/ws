package tunnel

import (
	"fmt"
	"net/netip"
	"slices"
)

// RoutePlan is the routes to delete and to add so that the routes in place are the networks wanted.
type RoutePlan struct {
	Want, Delete, Add []netip.Prefix
}

// PlanRoutes compares the networks wanted with the routes in place. It runs no route.
func PlanRoutes(want, routed []netip.Prefix) RoutePlan {
	p := RoutePlan{Want: want}
	for _, n := range routed {
		if !slices.Contains(want, n) {
			p.Delete = append(p.Delete, n)
		}
	}
	for _, n := range want {
		if !slices.Contains(routed, n) {
			p.Add = append(p.Add, n)
		}
	}
	return p
}

// Applied says what is routed once the adds ran, given those that failed, in the plan's order. A
// network that another route already takes cannot be routed: it is logged once, and tried again at
// each plan. unroutable holds the networks logged, and keeps those of out, which AllowedFor left
// out, so that trials that leave one out and take it back log it once. tun is the utun's name.
func (p RoutePlan) Applied(failed, out, unroutable []netip.Prefix, tun string) (routed, unroutableNow []netip.Prefix, logs []string) {
	for _, n := range out {
		if slices.Contains(unroutable, n) {
			unroutableNow = append(unroutableNow, n)
		}
	}
	for _, n := range p.Want {
		if !slices.Contains(failed, n) {
			routed = append(routed, n)
			continue
		}
		if !slices.Contains(unroutable, n) {
			logs = append(logs, fmt.Sprintf(
				"cannot route %s into %s, as another route to it may be in the way", n, tun))
		}
		unroutableNow = append(unroutableNow, n)
	}
	return routed, unroutableNow, logs
}
