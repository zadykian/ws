package values

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"
)

// check refuses the cable's values that the role's netplan file could not hold, where a port is
// set. It refuses a port that no interface's name takes, and an address that is no host's. The
// file quotes the port in YAML's double quotes. So check refuses each byte that would need an
// escape there, with the others that JSON escapes. Without a port, nothing is checked, as the role
// writes no file.
func (c Cable) check() error {
	if c.Port == "" {
		return nil
	}
	if strings.ContainsFunc(c.Port, func(r rune) bool {
		return r <= ' ' || r == 0x7f || strings.ContainsRune("/\"\\<>&", r)
	}) {
		return fmt.Errorf("WS_CABLE_PORT %q is not an interface's name", c.Port)
	}
	if problems := AddressProblems(c.Address); len(problems) > 0 {
		return errors.New("WS_CABLE_ADDRESS: " + strings.Join(problems, " "))
	}
	// AddressProblems has parsed it, but netip refuses a prefix length with a leading zero.
	subnet, err := netip.ParsePrefix(c.Address)
	if err != nil {
		return fmt.Errorf("WS_CABLE_ADDRESS %q is not an address with its prefix length, as 10.77.0.1/30: %w", c.Address, err)
	}
	if !inPrivateRanges(subnet.Masked()) {
		return fmt.Errorf("WS_CABLE_ADDRESS %s is outside the private ranges, which the guests' ACL drops: a guest would reach the host", c.Address)
	}
	return nil
}
