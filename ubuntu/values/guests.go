package values

import (
	"errors"
	"fmt"
	"net/netip"
	"regexp"
	"strings"
)

var (
	guestName = regexp.MustCompile(`^[a-z]([a-z0-9-]{0,10}[a-z0-9])?$`)
	// guestSize is a size that Incus reads as a whole number of binary units.
	guestSize = regexp.MustCompile(`^[1-9][0-9]*(MiB|GiB|TiB)$`)
)

// Guest is a guest that the host runs, with the limits that Incus holds it to. Its other values
// are the guest's own, in its file.
type Guest struct {
	// Name is the guest's name and its hostname. It reaches Incus's commands, a path, and the
	// secrets' profile, so it holds lowercase letters, digits and dashes, and starts with a letter.
	Name string `yaml:"name"`
	// Address is the guest's address on the bridge: its lease and the source it may send from.
	Address string `yaml:"address"`
	// CPU is the number of cores the guest may use.
	CPU int `yaml:"cpu"`
	// Memory is the guest's memory limit, as 8GiB, and Disk its root disk's quota, as 200GiB.
	Memory string `yaml:"memory"`
	Disk   string `yaml:"disk"`
}

// guestsCheck refuses a list that no guest could run by: each guest's fields, and no name or
// address twice. bridge is the bridge's address with its prefix length, whose subnet the guests share.
func guestsCheck(guests []Guest, bridge netip.Prefix) error {
	names, addrs := map[string]bool{}, map[netip.Addr]bool{}
	for i, g := range guests {
		if err := g.check(bridge); err != nil {
			return fmt.Errorf("WS_INCUS_GUESTS, guest %d: %w", i+1, err)
		}
		ip := netip.MustParseAddr(g.Address)
		switch {
		case names[g.Name]:
			return fmt.Errorf("WS_INCUS_GUESTS holds the name %s twice", g.Name)
		case addrs[ip]:
			return fmt.Errorf("WS_INCUS_GUESTS holds the address %s twice", g.Address)
		}
		names[g.Name], addrs[ip] = true, true
	}
	return nil
}

// check refuses a guest that Incus could not run. It names the field.
func (g Guest) check(bridge netip.Prefix) error {
	var problems []string
	if !guestName.MatchString(g.Name) {
		problems = append(problems, fmt.Sprintf("name %q is not 1 to 12 lowercase letters, digits and dashes, starting with a letter and not ending in a dash.", g.Name))
	}
	if p := g.addressProblem(bridge); p != "" {
		problems = append(problems, p)
	}
	if g.CPU < 1 {
		problems = append(problems, fmt.Sprintf("cpu is %d, not a number of cores from 1.", g.CPU))
	}
	for _, f := range []struct{ field, size string }{{"memory", g.Memory}, {"disk", g.Disk}} {
		if !guestSize.MatchString(f.size) {
			problems = append(problems, fmt.Sprintf("%s is %q, not a whole number of MiB, GiB or TiB, as 8GiB.", f.field, f.size))
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return errors.New(g.label() + strings.Join(problems, " "))
}

// label starts a guest's problems with the guest, where it has a name.
func (g Guest) label() string {
	if g.Name == "" {
		return ""
	}
	return g.Name + ": "
}

// addressProblem says why the address is no guest's on the bridge, or returns nothing. The bridge
// holds the first address, and the broadcast and the network's own are no host's.
func (g Guest) addressProblem(bridge netip.Prefix) string {
	ip, err := netip.ParseAddr(g.Address)
	switch {
	case err != nil || !ip.Is4():
		return fmt.Sprintf("address %q is not an IPv4 address, as 10.100.0.10.", g.Address)
	case !bridge.Masked().Contains(ip):
		return fmt.Sprintf("address %s is not in the bridge's subnet %s.", ip, bridge.Masked())
	case ip == bridge.Addr():
		return fmt.Sprintf("address %s is the bridge's own address.", ip)
	case !isHost(ip, bridge.Masked()):
		return fmt.Sprintf("address %s is not a host's address in %s.", ip, bridge.Masked())
	}
	return ""
}
