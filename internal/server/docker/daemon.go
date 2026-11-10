package docker

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/zadykian/ws/internal/server/values"
)

// bridgeCIDR is the default bridge's address with its prefix length.
func bridgeCIDR(d values.Docker) string {
	return d.BridgeIP + "/" + strconv.Itoa(d.BridgePrefixLength)
}

// buildCache returns the build cache's settings, with the cache's limits in its policy, and the age
// in hours, as Go has no unit of days.
func buildCache(d values.Docker, c Cache) builder {
	limits := gcPolicy{ReservedSpace: c.Reserved, MaxUsedSpace: c.MaxUsed, MinFreeSpace: c.MinFree}
	policy := []gcPolicy{limits, {All: true, ReservedSpace: c.Reserved, MaxUsedSpace: c.MaxUsed, MinFreeSpace: c.MinFree}}
	if d.BuildCacheKeepDays > 0 {
		age := gcPolicy{All: true, Filter: []string{fmt.Sprintf("until=%dh", d.BuildCacheKeepDays*24)}}
		policy = append([]gcPolicy{age}, policy...)
	}
	var b builder
	b.GC.Enabled = true
	b.GC.Policy = policy
	return b
}

// publishOpts makes a port published with no address listen on the values' address.
func publishOpts(d values.Docker) map[string]map[string]string {
	return map[string]map[string]string{"bridge": {"com.docker.network.bridge.host_binding_ipv4": d.PublishIP}}
}

// DaemonJSON returns the system daemon's daemon.json for the values. The file applies as dockerd
// restarts.
func DaemonJSON(d values.Docker, c Cache) ([]byte, error) {
	j := daemonJSON{
		Bip:                 bridgeCIDR(d),
		Builder:             buildCache(d, c),
		CgroupParent:        "docker.slice",
		DefaultAddressPools: []addressPool{{Base: d.AddressPool, Size: d.AddressPoolSize}},
		DefaultNetworkOpts:  publishOpts(d),
		DNS:                 []string{d.BridgeIP},
		IP:                  d.PublishIP,
		LiveRestore:         true,
		LogDriver:           "json-file",
		LogOpts:             logOpts(d),
	}
	out, err := json.MarshalIndent(j, "", "  ")
	return append(out, '\n'), err
}

func logOpts(d values.Docker) map[string]string {
	return map[string]string{"max-size": d.LogMaxSize, "max-file": strconv.Itoa(d.LogMaxFile)}
}

// slirpDNS is slirp4netns's own resolver, which answers through the host's. Containers can't use a
// copy of the host's servers. Where these are private, the output table drops what slirp4netns
// sends them on a container's behalf, while its resolver asks by the host's 127.0.0.53.
const slirpDNS = "10.0.2.3"

// RootlessJSON returns the rootless daemon's daemon.json. It has no bip or address pools, as its
// bridges live in rootlesskit's namespace and meet no route of the host's. Nor has it
// live-restore, which a rootless daemon accepts and does not honour. cgroup-parent keeps the containers inside
// docker.slice, as without it they land beside the unit and outside its limits.
func RootlessJSON(d values.Docker, c Cache) ([]byte, error) {
	j := rootlessJSON{
		Builder:            buildCache(d, c),
		CgroupParent:       "docker.slice",
		DefaultNetworkOpts: publishOpts(d),
		DNS:                []string{slirpDNS},
		IP:                 d.PublishIP,
		LogDriver:          "json-file",
		LogOpts:            logOpts(d),
	}
	out, err := json.MarshalIndent(j, "", "  ")
	return append(out, '\n'), err
}
