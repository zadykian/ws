// Package slot gives the tunnel of a server's slot, WS_SLOT. Slot N's subnet is 10.99.N.0/24, with
// the server at .1 and the Mac at .2. The Mac's side and the server's both read it.
package slot

import "net/netip"

// Last is the highest slot.
const Last = 15

// Range holds every slot's subnet. A LAN and a tunnel's routed networks stay clear of it.
var Range = netip.MustParsePrefix("10.99.0.0/20")

// Slot is a server's slot, 0 to Last. The zero value is slot 0, whose ends ws had before slots.
type Slot uint8

// Net is the slot's subnet.
func (s Slot) Net() netip.Prefix {
	a := Range.Addr().As4()
	a[2] += byte(s)
	return netip.PrefixFrom(netip.AddrFrom4(a), 24)
}

// Server is the server's end of the tunnel, the subnet's .1.
func (s Slot) Server() netip.Addr {
	return s.Net().Addr().Next()
}

// Mac is the Mac's end of the tunnel, the subnet's .2.
func (s Slot) Mac() netip.Addr {
	return s.Server().Next()
}
