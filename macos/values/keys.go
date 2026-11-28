package values

import (
	"strconv"

	"github.com/zadykian/ws/engine/core/check"
	"github.com/zadykian/ws/internal/cablenet"
	"github.com/zadykian/ws/internal/slot"
)

// defaultPort is the UDP port of wg0 where WS_TUNNEL_PORT is empty, and forwardPort the TCP port that a
// router forwards to ws's 22 where WS_REMOTE_PORT is.
const (
	defaultPort = 51820
	forwardPort = 22022
)

// Conf is values.yml's settings, each key's value as the file gives it, with the default of a key
// that has no line or an empty value.
type Conf struct {
	Target, Host, Slot, PublicKey, LAN, LANAddress, Remote, RemotePort, Port string
	TunnelNetworks, Resolvers, CableService, CablePort                       string

	// HostCable is set by ReadFleet alone, for a guest whose host has a cable: the guest tries the
	// cable's address too. A guest holds no WS_CABLE_SERVICE, as one cable serves one service.
	HostCable bool
}

// TunnelSlot gives the slot that c.Slot names. ReadFleet has checked it, and an empty one is slot 0.
func (c Conf) TunnelSlot() slot.Slot {
	s, _ := slot.Parse(c.Slot)
	return s
}

// TunnelPort gives the UDP port of wg0 on the server: WS_TUNNEL_PORT, or 51820 where it has none.
func (c Conf) TunnelPort() string {
	if c.Port == "" {
		return strconv.Itoa(defaultPort)
	}
	return c.Port
}

// ForwardPort gives the TCP port the router forwards to ws's 22: WS_REMOTE_PORT, or 22022 where it
// has none.
func (c Conf) ForwardPort() string {
	if c.RemotePort == "" {
		return strconv.Itoa(forwardPort)
	}
	return c.RemotePort
}

// key is one of values.yml's keys: where its value goes, its check, and the form its message names.
type key struct {
	name  string
	field func(*Conf) *string
	ok    func(value string) bool
	form  string
}

// keptNames names the networks that the cable and the tunnel keep, for the forms of WS_LAN and
// WS_TUNNEL_NETWORKS.
var keptNames = cablenet.Net.String() + " and " + slot.Range.String()

// port is the form of both ports.
const port = "a port, 1 to 65535"

// keys is values.yml's keys in the order of the file's docs, with their checks.
var keys = []key{
	{"WS_TARGET", func(c *Conf) *string { return &c.Target },
		func(v string) bool { return v == "lan" || v == "cloud" },
		"lan or cloud: a machine on the LAN or a cloud VM"},
	{"WS_HOST", func(c *Conf) *string { return &c.Host }, IsName,
		"a server's name, the host of this guest: " + NameForm},
	{"WS_SLOT", func(c *Conf) *string { return &c.Slot }, func(v string) bool { _, ok := slot.Parse(v); return ok },
		"a number from 0 to " + strconv.Itoa(slot.Last)},
	{"WS_PUBLIC_KEY", func(c *Conf) *string { return &c.PublicKey }, check.Key,
		"a WireGuard public key, 43 base64 characters and an ="},
	{"WS_LAN", func(c *Conf) *string { return &c.LAN }, isLAN,
		"an IPv4 network, as 10.88.0.0/23, clear of " + keptNames},
	{"WS_LAN_ADDRESS", func(c *Conf) *string { return &c.LANAddress }, check.IPv4,
		"a host's IPv4 address inside WS_LAN"},
	{"WS_REMOTE", func(c *Conf) *string { return &c.Remote }, check.Name,
		"a host name or an IPv4 address"},
	{"WS_REMOTE_PORT", func(c *Conf) *string { return &c.RemotePort }, check.Port, port},
	{"WS_TUNNEL_PORT", func(c *Conf) *string { return &c.Port }, check.Port, port},
	{"WS_TUNNEL_NETWORKS", func(c *Conf) *string { return &c.TunnelNetworks },
		func(v string) bool { return check.Networks(v, cablenet.Net, slot.Range) },
		"IPv4 networks split by blanks, as 10.20.0.0/16, each clear of " + keptNames},
	{"WS_RESOLVERS", func(c *Conf) *string { return &c.Resolvers }, check.Resolvers,
		"DOMAIN=ADDRESS[,ADDRESS...] split by blanks, as example.internal=10.20.0.53, each " +
			"domain once and in lower case, with up to 3 IPv4 addresses"},
	{"WS_CABLE_SERVICE", func(c *Conf) *string { return &c.CableService }, check.CableService,
		"a network service as networksetup -listallnetworkservices prints it, no * first"},
	{"WS_CABLE_PORT", func(c *Conf) *string { return &c.CablePort }, check.CablePort,
		"a port of ws's, as enp5s0: up to 15 letters, digits and ._-, no - first"},
}

// isLAN reports whether v is a network clear of the cable's and the slots' subnets.
func isLAN(v string) bool {
	p, ok := check.CIDR(v)
	return ok && !p.Overlaps(cablenet.Net) && !p.Overlaps(slot.Range)
}
