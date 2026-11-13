package ikeward

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/roles/server/ubuntu/internal/ufw"
)

// comment marks the ufw rules the role owns, as ufw show added prints them.
const comment = "ikeward"

// routes are the rules that let each peer's packets from wg0 out by ikeward0. Each names wg0, as a
// rule that forwards out ikeward0 from any interface would let in what ikeward's NAT rule takes,
// whatever its source.
func routes(peers []string) []ufw.Rule {
	var rules []ufw.Rule
	for _, p := range peers {
		rules = append(rules, ufw.Route("allow", "wg0", "ikeward0", p, comment))
	}
	return rules
}

// owns tells the rules the role made: route rules from a peer's address, wg0 to ikeward0. One made
// by hand without the comment is not read, and is taken as the role's when its peer is listed.
func owns(r ufw.Rule) bool {
	want := ufw.Route("allow", "wg0", "ikeward0", r.From, comment)
	return r == want
}

// forward returns the resources of the WireGuard peers' way to the VPN: the kernel forwards, and
// ufw lets each peer's packets through. ikeward's NAT rule gives them the address the VPN
// assigned. With no peer, the file goes, and so do the role's rules.
func forward(m effect.Machine, out effect.Out, peers []string) []host.Resource {
	return append(sysctl(m, len(peers) > 0), ufw.Rules{M: m, Out: out, Comment: comment, Want: routes(peers),
		Owns: owns, NeedsSystemd: true})
}
