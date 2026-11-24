package incus

import (
	"maps"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/ubuntu/roles/incus/object"
)

// ProfileName is the profile that each guest takes.
const ProfileName = "ws"

const profileDescription = "ws: an unprivileged guest on the bridge, with its disk in the pool"

// KernelModules are what a guest's Docker needs and a guest cannot load: Incus loads them on the
// host as the guest starts. br_netfilter stays out, as the bridge's table of sources replaces
// it. The tunnel's xfrm modules join them in #1530.
var KernelModules = []string{
	"overlay", "veth", "bridge", "ip_tables", "nf_nat", "nft_compat", "nft_chain_nat", "xt_conntrack", "xt_MASQUERADE",
	"xt_addrtype",
}

// profileKind is incus profile, whose edit takes the description, the config and the devices.
var profileKind = object.Kind{
	Noun:   "profile",
	List:   []string{"profile", "list"},
	Create: []string{"profile", "create", object.NameArg},
	Edit:   []string{"profile", "edit", object.NameArg},
	Keys:   []string{"description", "config", "devices"},
}

// Profile is what each guest takes: unprivileged, with its own range of IDs, and nesting for the
// Docker it runs. Its NIC joins the bridge with a MAC filter and port isolation, so a guest sends
// from its own MAC alone and reaches no other guest at layer 2. Its disk is on the pool.
func Profile(m effect.Machine) object.Object {
	return object.Object{
		M: m, Kind: profileKind, Name: ProfileName,
		Want: map[string]any{
			"description": profileDescription,
			"config": map[string]any{
				"security.privileged":     "false",
				"security.nesting":        "true",
				"security.idmap.isolated": "true",
				"linux.kernel_modules":    strings.Join(KernelModules, ","),
				"boot.autostart":          "true",
				"snapshots.schedule":      snapshotSchedule,
				"snapshots.expiry":        snapshotExpiry,
			},
			"devices": map[string]any{"eth0": nicDevice(nil), "root": rootDevice("")},
		},
	}
}

// A guest's disk is snapshotted every day, and a snapshot expires after two weeks. A schedule with
// no expiry keeps every snapshot, and a quota that counts them (see Pool) fills.
const (
	snapshotSchedule = "@daily"
	snapshotExpiry   = "2w"
)

// nicDevice is the profile's NIC, with the keys of extra laid over it. A guest's own NIC takes
// the profile's keys and adds its port's name and its address. A device of an instance replaces
// the profile's of the same name as a whole.
func nicDevice(extra map[string]any) map[string]any {
	nic := map[string]any{
		"type": "nic", "network": BridgeName, "name": "eth0",
		"security.mac_filtering": "true", "security.port_isolation": "true",
	}
	maps.Copy(nic, extra)
	return nic
}

// rootDevice is the profile's disk on the pool, with a quota where size is set: a guest's own.
func rootDevice(size string) map[string]any {
	root := map[string]any{"type": "disk", "path": "/", "pool": PoolName}
	if size != "" {
		root["size"] = size
	}
	return root
}
