package check

import (
	"net/netip"
	"strings"
)

// LANHost reports whether a is a host's address in the network lan: inside it, and neither its
// first address, the network's own, nor its last, its broadcast. In a /31 or a /32, every address
// is a host's.
func LANHost(lan netip.Prefix, a netip.Addr) bool {
	if !lan.Contains(a) {
		return false
	}
	last := uint32(1)<<(32-lan.Bits()) - 1
	b := a.As4()
	host := (uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])) & last
	return last < 3 || (host != 0 && host != last)
}

// Networks reports whether list is IPv4 networks split by blanks, each as CIDR takes one and none
// overlapping a network of reserved. Its characters are checked first, so that a list of any other
// kind fails alone.
func Networks(list string, reserved ...netip.Prefix) bool {
	if strings.Trim(list, "0123456789./ \t") != "" {
		return false
	}
	for _, field := range strings.Fields(list) {
		p, ok := CIDR(field)
		if !ok {
			return false
		}
		for _, c := range reserved {
			if p.Overlaps(c) {
				return false
			}
		}
	}
	return true
}
