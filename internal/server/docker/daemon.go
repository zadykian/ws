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

// DaemonJSON returns daemon.json for the values, with the cache's limits in the build cache's
// policy, and the age in hours, as Go has no unit of days. The file applies as dockerd restarts.
func DaemonJSON(d values.Docker, c Cache) ([]byte, error) {
	limits := gcPolicy{ReservedSpace: c.Reserved, MaxUsedSpace: c.MaxUsed, MinFreeSpace: c.MinFree}
	policy := []gcPolicy{limits, {All: true, ReservedSpace: c.Reserved, MaxUsedSpace: c.MaxUsed, MinFreeSpace: c.MinFree}}
	if d.BuildCacheKeepDays > 0 {
		age := gcPolicy{All: true, Filter: []string{fmt.Sprintf("until=%dh", d.BuildCacheKeepDays*24)}}
		policy = append([]gcPolicy{age}, policy...)
	}
	j := daemonJSON{
		Bip:                 bridgeCIDR(d),
		CgroupParent:        "docker.slice",
		DefaultAddressPools: []addressPool{{Base: d.AddressPool, Size: d.AddressPoolSize}},
		DefaultNetworkOpts: map[string]map[string]string{
			"bridge": {"com.docker.network.bridge.host_binding_ipv4": d.PublishIP},
		},
		DNS:         []string{d.BridgeIP},
		IP:          d.PublishIP,
		LiveRestore: true,
		LogDriver:   "json-file",
		LogOpts:     map[string]string{"max-size": d.LogMaxSize, "max-file": strconv.Itoa(d.LogMaxFile)},
	}
	j.Builder.GC.Enabled = true
	j.Builder.GC.Policy = policy
	out, err := json.MarshalIndent(j, "", "  ")
	return append(out, '\n'), err
}
