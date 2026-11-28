package routing

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	tunnel "github.com/zadykian/ws/macos/roles/tunnel/plan"
)

// Ran is what RunRoutes did. Failed holds the networks it could not add, in the plan's order,
// and Rejects the -reject routes that the daemon left once it ran. Logs hold a line for each
// route it could not add or delete, or left alone.
type Ran struct {
	Failed, Rejects []netip.Prefix
	Logs            []string
}

// RunRoutes carries out a plan into the utun tun, given the daemon's rejects. It deletes first, as
// an old route may be in the way of a new one. A reject comes before a delete, or right after the
// delete at its own prefix, and the unrejects go last: one network goes in clear for a step.
func RunRoutes(ctx context.Context, m effect.Machine, plan tunnel.RoutePlan, tun string, rejects []netip.Prefix) Ran {
	r := runner{ctx: ctx, m: m, tun: tun, ran: Ran{Rejects: slices.Clone(rejects)}}
	for _, y := range plan.Reject {
		if !slices.Contains(plan.Delete, y) {
			r.reject(y)
		}
	}
	for _, n := range plan.Delete {
		r.delete(n)
		if slices.Contains(plan.Reject, n) {
			r.reject(n)
		}
	}
	for _, n := range plan.Add {
		if !r.route("add", n, "-interface", tun) && !r.replace(n) {
			r.ran.Failed = append(r.ran.Failed, n)
		}
	}
	for _, y := range plan.Unreject {
		// A reject that a failed take-back put back stays, as its network would go in clear.
		if !slices.Contains(r.ran.Failed, y) {
			r.unreject(y)
		}
	}
	return r.ran
}

// runner runs the routes of one plan, and keeps what it did.
type runner struct {
	ctx context.Context
	m   effect.Machine
	tun string
	ran Ran
}

// route runs route's verb for the network n, with args after it, and tells whether it passed.
func (r *runner) route(verb string, n netip.Prefix, args ...string) bool {
	_, err := r.m.Run(r.ctx, "route", append([]string{"-q", "-n", verb, "-inet", n.String()}, args...)...)
	return err == nil
}

// get returns what route -n get says of the best route to n, and false where none answers.
func (r *runner) get(n netip.Prefix) (answer, bool) {
	out, err := r.m.Run(r.ctx, "route", "-n", "get", "-inet", n.String())
	if err != nil {
		return answer{}, false
	}
	return answerOf(string(out)), true
}

// delete deletes the route to n where it goes through tun alone: another server's daemon may
// route the network through its own utun.
func (r *runner) delete(n netip.Prefix) {
	a, ok := r.get(n)
	if !ok || a.route != n {
		// route get answers with the best match, as a wider route or the default, where the
		// network's own is gone. That route is not the network's to delete.
		return
	}
	if a.iface != r.tun {
		r.log("leaves the route to %s alone: it goes through %s, not %s", n, a.iface, r.tun)
		return
	}
	if !r.route("delete", n) {
		r.log("cannot remove the route to %s", n)
	}
}

func (r *runner) log(format string, a ...any) {
	r.ran.Logs = append(r.ran.Logs, fmt.Sprintf(format, a...))
}

// answer is what route -n get says of a network's best route: its network, from its destination
// and mask, its interface, and whether it rejects.
type answer struct {
	route  netip.Prefix
	iface  string
	reject bool
}

// answerOf reads route -n get's answer. A host route's answer may hold no mask line, so a
// destination with none is a /32. The network is invalid where the answer gives none, as the
// default's "default" is none.
func answerOf(out string) (a answer) {
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
			a.iface = v
		case "flags":
			a.reject = slices.Contains(strings.Split(strings.Trim(v, "<>"), ","), "REJECT")
		}
	}
	bits := 32
	if masked {
		ones, size := net.IPMask(mask.AsSlice()).Size()
		if size == 0 {
			return a // a mask that is no address, or not of leading ones
		}
		bits = ones
	}
	if dest.Is4() {
		a.route = netip.PrefixFrom(dest, bits)
	}
	return a
}
