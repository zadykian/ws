package ikeward

import (
	_ "embed" // sysctlFile

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

const sysctlPath = "/etc/sysctl.d/99-ikeward.conf"

//go:embed 99-ikeward.conf
var sysctlFile []byte

// sysctl returns the resources of IPv4 forwarding at boot, or the removal of the file
// where none is wanted.
func sysctl(m effect.Machine, want bool) []host.Resource {
	if !want {
		return []host.Resource{host.Absent{M: m, Path: sysctlPath}}
	}
	return []host.Resource{host.File{M: m, Path: sysctlPath, Content: sysctlFile, Mode: 0o644}}
}
