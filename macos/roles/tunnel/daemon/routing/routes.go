package routing

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	tunnel "github.com/zadykian/ws/macos/roles/tunnel/plan"
)

// Ran is what RunRoutes did. Failed holds the networks it could not add, in the plan's order,
// and Logs a line for each route it could not delete or left alone.
type Ran struct {
	Failed []netip.Prefix
	Logs   []string
}

// RunRoutes carries out a plan of routes into the utun tun. It deletes first, as an old route may
// be in the way of a new one.
func RunRoutes(ctx context.Context, m effect.Machine, plan tunnel.RoutePlan, tun string) Ran {
	r := runner{ctx: ctx, m: m, tun: tun}
	for _, n := range plan.Delete {
		r.delete(n)
	}
	for _, n := range plan.Add {
		if !r.route("add", n, "-interface", tun) {
			r.ran.Failed = append(r.ran.Failed, n)
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
// and mask, and its interface.
type answer struct {
	route netip.Prefix
	iface string
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
