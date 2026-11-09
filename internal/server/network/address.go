// Package network sets up the server's end of the cable to the Mac: an address on the cable's
// port, from a netplan file of its own. The checks that come before the file are pure.
package network

import (
	"fmt"
	"net/netip"
	"strconv"
	"strings"
)

// longestBroadcastPrefix is the longest prefix of a subnet of more than two addresses, which keeps
// its first and its last for itself and its broadcast. A /31's two addresses are both hosts'
// (RFC 3021), and a /32 is one host's alone.
const longestBroadcastPrefix = 30

// AddressProblems returns why value is not an IPv4 address of a host with its prefix length, as
// 10.77.0.1/30. The list is empty for such an address. A subnet of more than two addresses keeps its
// first and its last for itself and its broadcast, so neither is a host's.
func AddressProblems(value string) []string {
	address, prefix, found := strings.Cut(value, "/")
	if !found || !digits(prefix) {
		return []string{value + " is not an IPv4 address with its prefix length, as 10.77.0.1/30"}
	}
	ip, err := netip.ParseAddr(address)
	if err != nil || !ip.Is4() {
		return []string{fmt.Sprintf("%s is not an IPv4 address with its prefix length: %q is not an IPv4 address", value, address)}
	}
	bits, err := strconv.Atoi(prefix)
	if err != nil || bits > 32 {
		return []string{fmt.Sprintf("%s is not an IPv4 address with its prefix length: %q is not a prefix length of 0 to 32", value, prefix)}
	}
	subnet := netip.PrefixFrom(ip, bits).Masked()
	if bits <= longestBroadcastPrefix && ip == subnet.Addr() {
		return []string{fmt.Sprintf("%s is the address of the subnet %s itself, not of a host in it", address, subnet)}
	}
	if bits <= longestBroadcastPrefix && ip == broadcast(subnet) {
		return []string{fmt.Sprintf("%s is the broadcast address of %s, not a host's", address, subnet)}
	}
	return nil
}

// digits reports whether s is one or more ASCII digits.
func digits(s string) bool {
	return s != "" && strings.Trim(s, "0123456789") == ""
}

// broadcast returns the last address of an IPv4 subnet.
func broadcast(subnet netip.Prefix) netip.Addr {
	last := subnet.Addr().As4()
	for bit := subnet.Bits(); bit < 32; bit++ {
		last[bit/8] |= 0x80 >> (bit % 8)
	}
	return netip.AddrFrom4(last)
}
