package incus

import (
	"errors"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/ubuntu/roles/incus/object"
	"github.com/zadykian/ws/ubuntu/values"
)

// BridgeName is the bridge that the guests' NICs join.
const BridgeName = "incusbr0"

// bridgeKind is incus network, whose edit takes the description and the config of a network.
var bridgeKind = object.Kind{
	Noun:   "network",
	List:   []string{"network", "list"},
	Create: []string{"network", "create", object.NameArg},
	Edit:   []string{"network", "edit", object.NameArg},
	Keys:   []string{"description", "config"},
}

// Bridge is the managed bridge incusbr0: NATed, with no IPv6, and its DHCP handing out the
// resolvers of WS_INCUS_BRIDGE_DNS, not the host's. The egress ACL is on it, and egress is
// dropped but for what that ACL allows. A name that another network holds is refused, as an edit
// would turn that network into the guests'.
func Bridge(m effect.Machine, v values.Incus) object.Object {
	return object.Object{
		M: m, Kind: bridgeKind, Name: BridgeName, Args: []string{"--type", "bridge"},
		Want: map[string]any{
			"description": "ws's bridge for the guests",
			"config": map[string]any{
				"ipv4.address":                        v.BridgeAddress,
				"ipv4.nat":                            "true",
				"ipv6.address":                        "none",
				"dns.nameservers":                     strings.Join(v.BridgeDNS, ","),
				"security.acls":                       ACLName,
				"security.acls.default.egress.action": "drop",
			},
		},
		Check: func(have map[string]any) error {
			if have["managed"] != true || have["type"] != "bridge" {
				return errors.New("is not a managed bridge: ws never turns another network into the guests'")
			}
			return nil
		},
	}
}
