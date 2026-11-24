package incus

import (
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/ubuntu/roles/incus/object"
	"github.com/zadykian/ws/ubuntu/values"
)

// IngressACLName is the ACL of what reaches the guests, on the bridge beside the egress ACL.
const IngressACLName = "ws-ingress"

const (
	ingressDescription = "ws: a tunnel's peer reaches its guest, and a forwarding guest's peer reaches any"
	toGuestDescription = "a peer reaches its guest's own addresses"
	toAnyDescription   = "a peer that the guest forwards for reaches any destination"
)

// IngressACL holds one allow for each peer of each guest's tunnel. A peer that the guest's ikeward
// forwards for, one of WS_IKEWARD_FORWARD_FROM, reaches any destination, as ikeward keeps no rule
// of its own. Any other peer reaches the guest's own addresses: its address on the bridge and the
// slot's. The bridge's default for ingress is drop, so the ACL holds allows alone.
func IngressACL(m effect.Machine, tunnels []values.GuestTunnel) object.Object {
	rules := []any{}
	for _, t := range tunnels {
		for _, p := range t.Peers {
			rule := map[string]any{"action": "allow", "state": "enabled", "source": p.Address}
			if slices.Contains(t.ForwardFrom, p.Address) {
				rule["description"] = t.Guest.Name + ": " + toAnyDescription
			} else {
				rule["destination"] = strings.Join([]string{t.Guest.Address, t.SlotAddress().String()}, ",")
				rule["description"] = t.Guest.Name + ": " + toGuestDescription
			}
			rules = append(rules, rule)
		}
	}
	return object.Object{
		M: m, Kind: aclKind, Name: IngressACLName,
		Want: map[string]any{"description": ingressDescription, "ingress": rules},
	}
}
