package incus

import (
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/ubuntu/roles/incus/object"
	"github.com/zadykian/ws/ubuntu/values"
)

// ACLName is the egress ACL that the bridge carries.
const ACLName = "ws-egress"

const (
	aclDescription   = "ws: a guest reaches the internet, and none of the private ranges"
	dropDescription  = "the private ranges, the host's addresses among them"
	allowDescription = "the rest, which is the internet"
)

// aclKind is incus network acl, whose edit takes these keys of an ACL (howto/network_acls).
var aclKind = object.Kind{
	Noun:   "network ACL",
	List:   []string{"network", "acl", "list"},
	Create: []string{"network", "acl", "create", object.NameArg},
	Edit:   []string{"network", "acl", "edit", object.NameArg},
	Keys:   []string{"description", "ingress", "egress", "config"},
}

// EgressACL drops what a guest sends to values.PrivateRanges and allows the rest. Incus applies each
// drop before any allow, so the order of the rules means nothing. The bridge's default for egress
// is drop, so a guest reaches what an allow names alone. Ingress is the tunnel's, not this one's.
func EgressACL(m effect.Machine) object.Object {
	return object.Object{
		M: m, Kind: aclKind, Name: ACLName,
		Want: map[string]any{
			"description": aclDescription,
			"egress": []any{
				map[string]any{"action": "drop", "state": "enabled", "destination": strings.Join(values.PrivateRanges, ","), "description": dropDescription},
				map[string]any{"action": "allow", "state": "enabled", "description": allowDescription},
			},
		},
	}
}
