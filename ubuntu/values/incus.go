package values

import (
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"strings"

	"github.com/zadykian/ws/internal/cablenet"
)

// Incus is roles/incus's keys, which a host reads alone.
type Incus struct {
	// PoolSource is the partition that the ZFS pool takes whole, as /dev/disk/by-partuuid/ID. Empty,
	// ws makes no pool, and says so where it would.
	PoolSource string `yaml:"WS_INCUS_POOL_SOURCE"`
	// BridgeAddress is the bridge's address with its prefix length. Its subnet is the guests'.
	BridgeAddress string `yaml:"WS_INCUS_BRIDGE_ADDRESS"`
	// BridgeDNS are the resolvers that the bridge's DHCP hands to the guests.
	BridgeDNS []string `yaml:"WS_INCUS_BRIDGE_DNS"`
	// Guests are the guests that the host runs, each with its address and limits.
	Guests []Guest `yaml:"WS_INCUS_GUESTS"`
}

// PrivateRanges are the IPv4 destinations that a guest never reaches: the private and shared
// ranges, link-local, multicast and the limited broadcast. The egress ACL drops them. So the host's
// own addresses must sit among them, or a guest would reach the host's services.
var PrivateRanges = []string{
	"10.0.0.0/8", "100.64.0.0/10", "169.254.0.0/16", "172.16.0.0/12", "192.168.0.0/16", "224.0.0.0/4",
	"255.255.255.255/32",
}

// inPrivateRanges reports whether the whole of subnet lies in one of PrivateRanges.
func inPrivateRanges(subnet netip.Prefix) bool {
	return slices.ContainsFunc(PrivateRanges, func(r string) bool {
		p := netip.MustParsePrefix(r)
		return p.Bits() <= subnet.Bits() && p.Contains(subnet.Addr())
	})
}

// tunnelSlots is the range of the tunnel's slots, which no guest's subnet may overlap: the table of
// sources drops a guest's claim on it.
const tunnelSlots = "10.99.0.0/20"

// BridgeSubnet is the bridge's subnet, as 10.100.0.0/24, which its address gives. A template reads
// it, so a file names the range that the values set.
func (i Incus) BridgeSubnet() string {
	return netip.MustParsePrefix(i.BridgeAddress).Masked().String()
}

// Sources are the addresses that the guests may send from, each with its port, in the guests' order.
func (i Incus) Sources() []Source {
	var sources []Source
	for _, g := range i.Guests {
		sources = append(sources, g.Sources()...)
	}
	return sources
}

// TunnelSlots is the range of the tunnel's slots, which a guest may not claim as its source.
func (Incus) TunnelSlots() string { return tunnelSlots }

// defaultIncus is what a host runs with where its file names none: a bridge at 10.100.0.1/24, and
// public resolvers.
func defaultIncus() Incus {
	return Incus{BridgeAddress: "10.100.0.1/24", BridgeDNS: []string{"9.9.9.9", "1.1.1.1"}}
}

// check refuses the values that the role's objects could not hold. An unset partition passes: the
// pool's plan names it where it must make one.
func (i Incus) check() error {
	if i.PoolSource != "" && (!strings.HasPrefix(i.PoolSource, "/dev/") ||
		strings.ContainsFunc(i.PoolSource, func(r rune) bool { return r <= ' ' || r == 0x7f })) {
		return fmt.Errorf("WS_INCUS_POOL_SOURCE %q is not a device's path, as /dev/disk/by-partuuid/ID", i.PoolSource)
	}
	if problems := AddressProblems(i.BridgeAddress); len(problems) > 0 {
		return errors.New("WS_INCUS_BRIDGE_ADDRESS: " + strings.Join(problems, " "))
	}
	parsed, err := netip.ParsePrefix(i.BridgeAddress)
	if err != nil {
		return fmt.Errorf("WS_INCUS_BRIDGE_ADDRESS %q is not an address with its prefix length, as 10.100.0.1/24: %w", i.BridgeAddress, err)
	}
	bridge := parsed.Masked()
	if !inPrivateRanges(bridge) {
		return fmt.Errorf("WS_INCUS_BRIDGE_ADDRESS %s is outside the private ranges, which the guests' ACL drops: a guest would reach the host", i.BridgeAddress)
	}
	if slots := netip.MustParsePrefix(tunnelSlots); bridge.Overlaps(slots) {
		return fmt.Errorf("WS_INCUS_BRIDGE_ADDRESS %s overlaps %s, the tunnel's slots", i.BridgeAddress, tunnelSlots)
	}
	if bridge.Overlaps(cablenet.Net) {
		return fmt.Errorf("WS_INCUS_BRIDGE_ADDRESS %s overlaps %s, the cable's subnet", i.BridgeAddress, cablenet.Net)
	}
	if len(i.BridgeDNS) == 0 {
		return errors.New("WS_INCUS_BRIDGE_DNS is empty: the guests need a resolver, as 9.9.9.9")
	}
	seen := map[netip.Addr]bool{}
	for _, s := range i.BridgeDNS {
		ip, err := netip.ParseAddr(s)
		if err != nil || !ip.Is4() || !ip.IsGlobalUnicast() {
			return fmt.Errorf("WS_INCUS_BRIDGE_DNS holds %q: each takes one IPv4 address, as 9.9.9.9", s)
		}
		if seen[ip] {
			return fmt.Errorf("WS_INCUS_BRIDGE_DNS holds %s twice", s)
		}
		seen[ip] = true
		if inPrivateRanges(netip.PrefixFrom(ip, 32)) {
			return fmt.Errorf("WS_INCUS_BRIDGE_DNS holds %s, a private address: the egress ACL drops it, so no guest could resolve", s)
		}
	}
	return guestsCheck(i.Guests, parsed)
}
