// Package cablenet fixes the cable's /30 once, for both sides: the server's end is its .1 and the
// Mac's is its .2. No key sets it, so the Mac dials the address that the server holds.
package cablenet

import (
	"encoding/binary"
	"net/netip"
)

// Net is the cable's subnet.
var Net = netip.MustParsePrefix("10.77.0.0/30")

// Server is the server's end of the cable, the subnet's .1.
func Server() netip.Addr {
	return Net.Addr().Next()
}

// ServerPrefix is the server's address on the cable, with the subnet's length: 10.77.0.1/30.
func ServerPrefix() netip.Prefix {
	return netip.PrefixFrom(Server(), Net.Bits())
}

// Mac is the Mac's end of the cable, the subnet's .2.
func Mac() netip.Addr {
	return Server().Next()
}

// Mask is the subnet's mask in dotted form, as networksetup takes it: 255.255.255.252.
func Mask() string {
	var m [4]byte
	binary.BigEndian.PutUint32(m[:], ^uint32(0)<<(32-Net.Bits()))
	return netip.AddrFrom4(m).String()
}
