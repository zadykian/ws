package base

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/host"
)

// swappinessParts sets the kernel's swappiness at boot and in force. The value is set whatever the
// swap file's size: with no swap area, the kernel reclaims only the page cache, and the value counts
// again once one is back.
func swappinessParts(m effect.Machine, out effect.Out, value int) []host.Resource {
	return []host.Resource{
		sysctlFile{M: m, Swappiness: value},
		inForce{M: m, Out: out, Swappiness: value},
	}
}
