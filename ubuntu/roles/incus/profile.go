package incus

import (
	"maps"
	"strconv"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/ubuntu/roles/incus/object"
	"github.com/zadykian/ws/ubuntu/values"
)

// ProfileName is the profile that each guest takes.
const ProfileName = "ws"

const profileDescription = "ws: an unprivileged guest on the bridge, with its disk in the pool"

// KernelModules are what a guest's Docker needs and a guest cannot load: Incus loads them on the
// host as the guest starts. br_netfilter stays out, as the bridge's table of sources replaces
// it. The xfrm modules and nft_masq are what ikeward needs: strongSwan's ESP tunnel, its xfrm link
// and its masquerade.
var KernelModules = []string{
	"overlay", "veth", "bridge", "ip_tables", "nf_nat", "nft_compat", "nft_chain_nat", "xt_conntrack", "xt_MASQUERADE",
	"xt_addrtype", "xfrm_user", "xfrm_algo", "esp4", "xfrm_interface", "nft_masq",
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
// from its own MAC alone and reaches no other guest at layer 2. Its disk is on the pool. Its proxy
// device carries the collector's telemetry to the host's SigNoz.
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
			"devices": map[string]any{"eth0": nicDevice(nil), "root": rootDevice(""), "otlp": otlpDevice()},
		},
	}
}

// A guest's disk is snapshotted every day, and a snapshot expires after two weeks. A schedule with
// no expiry keeps every snapshot, which fill the pool's space outside the quotas (see Pool).
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

// proxyUser is the user and the group that the proxy runs as on the host, in place of root: nobody.
const proxyUser = "65534"

// otlpDevice is the link from a guest's collector to the host's SigNoz. The device listens in the
// guest on a Unix socket, which only otelcol-contrib may open, and connects from the host to the
// intake on its loopback. No ACL or ufw rule sees it: the guest's connection ends in the guest, and
// the host's side is a new one on lo. The device's ids and mode are the socket's. Its defaults, 0, 0
// and 0644, would admit root alone, as a connect needs write, and otelcol-contrib is not root.
// security.uid and security.gid keep the proxy from running as the host's root.
func otlpDevice() map[string]any {
	id := strconv.Itoa(values.OtelcolID)
	return map[string]any{
		"type": "proxy", "bind": "instance",
		"listen": "unix:" + values.OTLPSocket, "connect": "tcp:" + values.SignozIntake,
		"uid": id, "gid": id, "mode": "0600",
		"security.uid": proxyUser, "security.gid": proxyUser,
	}
}
