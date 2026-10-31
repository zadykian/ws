package tunnel

import (
	"fmt"
	"net/netip"
	"strconv"
	"strings"
)

// The tunnel's two ends and its subnet, and the cable's subnet and ws's end of it.
var (
	Server      = netip.MustParseAddr("10.99.0.1")
	Address     = netip.MustParseAddr("10.99.0.2")
	TunnelNet   = netip.MustParsePrefix("10.99.0.0/24")
	CableNet    = netip.MustParsePrefix("10.77.0.0/30")
	CableServer = netip.MustParseAddr("10.77.0.1")
)

// DefaultPort is WireGuard's, for a tunnel.conf with no WS_PORT.
const DefaultPort = 51820

// Conf is tunnel.conf: the KEY=value lines that setup copies from mac.conf.
type Conf struct {
	Key       string // ws's public key
	LAN       netip.Prefix
	LANServer netip.Addr
	Remote    string // a name or an address, resolved anew at each try
	Port      uint16
}

// ParseConf reads the text of tunnel.conf, which it never runs. It fails on a line it does not
// take, or one with a value that fails its check, and on a file with no WS_PUBLIC_KEY. A later line
// of a key replaces an earlier one.
func ParseConf(text string) (Conf, error) {
	c := Conf{Port: DefaultPort}
	for _, line := range strings.Split(text, "\n") {
		line = strings.Trim(line, " \t")
		if line == "" || line[0] == '#' {
			continue
		}
		name, value, found := strings.Cut(line, "=")
		if !found || !c.set(name, value) {
			return Conf{}, fmt.Errorf("tunnel.conf: cannot take '%s'", line)
		}
	}
	if c.Key == "" {
		return Conf{}, fmt.Errorf("tunnel.conf has no WS_PUBLIC_KEY")
	}
	return c, nil
}

// set takes one value, and reports whether its name is one of tunnel.conf's and the value passes
// its check.
func (c *Conf) set(name, value string) bool {
	switch name {
	case "WS_PUBLIC_KEY":
		c.Key = value
		return isKey(value)
	case "WS_LAN":
		p, ok := parseCIDR(value)
		c.LAN = p
		return ok
	case "WS_LAN_ADDRESS":
		a, err := netip.ParseAddr(value)
		c.LANServer = a
		return err == nil && a.Is4()
	case "WS_REMOTE":
		c.Remote = value
		return isName(value)
	case "WS_PORT":
		n, err := strconv.ParseUint(value, 10, 16)
		c.Port = uint16(n)
		return isPort(value) && err == nil
	}
	return false
}
