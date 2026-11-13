package bridge

import (
	"net"
	"net/netip"
)

// Addrs returns docker0's IPv4 addresses with their prefix lengths, and none where docker0
// is missing, as before Docker's first start since boot.
func Addrs() ([]netip.Prefix, error) { return addrsOf("docker0") }

func addrsOf(name string) ([]netip.Prefix, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, ifc := range ifaces {
		if ifc.Name != name {
			continue
		}
		addrs, err := ifc.Addrs()
		if err != nil {
			return nil, err
		}
		var held []netip.Prefix
		for _, a := range addrs {
			if p, err := netip.ParsePrefix(a.String()); err == nil && p.Addr().Is4() {
				held = append(held, p)
			}
		}
		return held, nil
	}
	return nil, nil
}
