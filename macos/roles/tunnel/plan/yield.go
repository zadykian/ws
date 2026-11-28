package plan

import (
	"fmt"
	"net/netip"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/check"
)

// ParseYield reads the networks that ws macos tunnel yield gives to another owner, a network a
// line, each once and in the file's order. A daemon refuses no yield: it skips a line that is no
// IPv4 network, with a warning.
func ParseYield(text string) (nets []netip.Prefix, warnings []string) {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		p, ok := check.CIDR(line)
		if !ok {
			warnings = append(warnings, fmt.Sprintf("the yield's line '%s' is no IPv4 network; skips it", line))
			continue
		}
		if !slices.Contains(nets, p) {
			nets = append(nets, p)
		}
	}
	return nets, warnings
}

// Yield returns the networks of own that no yielded network holds, and a warning for each it
// leaves out. A network holds those equal to it or narrower. A wider one of own stays, as the
// other owner's narrower routes win over it by length.
func Yield(own, yielded []netip.Prefix) (kept []netip.Prefix, warnings []string) {
	for _, n := range own {
		y, ok := heldBy(n, yielded)
		if !ok {
			kept = append(kept, n)
			continue
		}
		warnings = append(warnings, fmt.Sprintf("leaves %s out, as it yields %s to another owner", n, y))
	}
	return kept, warnings
}

// Holders returns the yielded networks that hold a network of own, equal or wider, each once. A
// yield leaves its -reject routes at them, even for a network that is out of the routes already.
func Holders(own, yielded []netip.Prefix) (held []netip.Prefix) {
	for _, n := range own {
		if y, ok := heldBy(n, yielded); ok && !slices.Contains(held, y) {
			held = append(held, y)
		}
	}
	return held
}

// WithRejects adds a -reject route at each network of held, from Holders. A narrower reject would
// win over the other owner's routes by length. The daemon's rejects are not added again, and each
// that the yield no longer holds goes.
func (p RoutePlan) WithRejects(held, yield, rejects []netip.Prefix) RoutePlan {
	for _, y := range held {
		if !slices.Contains(rejects, y) && !slices.Contains(p.Reject, y) {
			p.Reject = append(p.Reject, y)
		}
	}
	for _, y := range rejects {
		if !slices.Contains(yield, y) {
			p.Unreject = append(p.Unreject, y)
		}
	}
	return p
}

// heldBy returns the narrowest network of nets that holds n, equal to it or wider.
func heldBy(n netip.Prefix, nets []netip.Prefix) (netip.Prefix, bool) {
	var best netip.Prefix
	for _, y := range nets {
		if y.Bits() <= n.Bits() && y.Contains(n.Addr()) && (!best.IsValid() || y.Bits() > best.Bits()) {
			best = y
		}
	}
	return best, best.IsValid()
}
