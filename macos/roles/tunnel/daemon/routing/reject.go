package routing

import (
	"net/netip"
	"slices"
)

// reject adds a -reject route at y. One that stands there already, the other owner's or one
// that the daemon left before, stays, and the daemon takes it as its own.
func (r *runner) reject(y netip.Prefix) {
	if r.route("add", y, "127.0.0.1", "-reject") {
		r.own(y)
		return
	}
	if a, ok := r.get(y); ok && a.route == y && a.reject {
		r.own(y)
		return
	}
	r.log("cannot add a -reject route to %s, so it may go in clear until the other owner routes it", y)
}

// replace takes a -reject route that stands at n, whoever added it, for n's route into the utun.
// Where that add fails, it puts the reject back, as n would go in clear.
func (r *runner) replace(n netip.Prefix) bool {
	a, ok := r.get(n)
	if !ok || a.route != n || !a.reject || !r.route("delete", n) {
		return false
	}
	r.disown(n)
	if r.route("add", n, "-interface", r.tun) {
		return true
	}
	if r.route("add", n, "127.0.0.1", "-reject") {
		r.own(n)
	} else {
		r.log("cannot put the -reject route to %s back, so it may go in clear", n)
	}
	return false
}

// unreject deletes the -reject route that the daemon left at y, where it still stands.
func (r *runner) unreject(y netip.Prefix) {
	a, ok := r.get(y)
	if ok && a.route == y && a.reject && !r.route("delete", y) {
		r.log("cannot remove the -reject route to %s", y)
		return
	}
	r.disown(y)
}

func (r *runner) own(y netip.Prefix) {
	if !slices.Contains(r.ran.Rejects, y) {
		r.ran.Rejects = append(r.ran.Rejects, y)
	}
}

func (r *runner) disown(y netip.Prefix) {
	r.ran.Rejects = slices.DeleteFunc(r.ran.Rejects, func(p netip.Prefix) bool { return p == y })
}
