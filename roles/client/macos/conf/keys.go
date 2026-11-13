package conf

import (
	"strconv"

	"github.com/zadykian/ws/engine/core/check"
	tunnel "github.com/zadykian/ws/roles/client/macos/tunnel/plan"
	"github.com/zadykian/ws/roles/internal/slot"
)

// Conf is mac.conf's settings, each key's value as the file gives it, with the default of a key
// that has no line or an empty value.
type Conf struct {
	User, Target, Slot, PublicKey, LAN, LANAddress, Remote, RemotePort, Port string
	TunnelNetworks, Resolvers, CableService, CablePort                       string
	GitName, GitEmail, GitSigningPubKey, IdentityPubKey, IdentityAgent       string
}

// key is one of mac.conf's keys: where its value goes, its check, and the form its message names.
// A check gets whether the value was in quotes.
type key struct {
	name  string
	field func(*Conf) *string
	ok    func(value string, quoted bool) bool
	form  string
}

// plain adapts a check that does not care for quotes.
func plain(f func(string) bool) func(string, bool) bool {
	return func(v string, _ bool) bool { return f(v) }
}

// keptNames names the networks that the cable and the tunnel keep, for the forms of WS_LAN and
// WS_TUNNEL_NETWORKS.
var keptNames = tunnel.CableNet.String() + " and " + slot.Range.String()

// port is the form of both ports.
const port = "a port, 1 to 65535"

// keyFile is the form of both keys' files.
const keyFile = "a public key's file, as ~/.ssh/NAME.pub: a path from / or ~/ to .pub, " +
	"of letters, digits and ._/@+-"

// keys is mac.conf's keys in the order of the file's docs, with their checks.
var keys = []key{
	{"WS_USER", func(c *Conf) *string { return &c.User }, plain(check.User),
		"a user name of 32 characters at most: letters, digits, ., _ and -, no - first"},
	{"WS_TARGET", func(c *Conf) *string { return &c.Target },
		plain(func(v string) bool { return v == "lan" || v == "cloud" || v == "mac-vm" }),
		"lan, cloud or mac-vm: a machine on the LAN, a cloud VM or a VM on the Mac"},
	{"WS_SLOT", func(c *Conf) *string { return &c.Slot }, plain(func(v string) bool { _, ok := slot.Parse(v); return ok }),
		"a number from 0 to " + strconv.Itoa(slot.Last)},
	{"WS_PUBLIC_KEY", func(c *Conf) *string { return &c.PublicKey }, plain(check.Key),
		"a WireGuard public key, 43 base64 characters and an ="},
	{"WS_LAN", func(c *Conf) *string { return &c.LAN }, plain(isLAN),
		"an IPv4 network, as 10.88.0.0/24, clear of " + keptNames},
	{"WS_LAN_ADDRESS", func(c *Conf) *string { return &c.LANAddress }, plain(check.IPv4),
		"a host's IPv4 address inside WS_LAN"},
	{"WS_REMOTE", func(c *Conf) *string { return &c.Remote }, plain(check.Name),
		"a host name or an IPv4 address"},
	{"WS_REMOTE_PORT", func(c *Conf) *string { return &c.RemotePort }, plain(check.Port), port},
	{"WS_PORT", func(c *Conf) *string { return &c.Port }, plain(check.Port), port},
	{"WS_TUNNEL_NETWORKS", func(c *Conf) *string { return &c.TunnelNetworks },
		plain(func(v string) bool { return check.Networks(v, tunnel.CableNet, slot.Range) }),
		"IPv4 networks split by blanks, as 10.20.0.0/16, each clear of " + keptNames},
	{"WS_RESOLVERS", func(c *Conf) *string { return &c.Resolvers }, plain(check.Resolvers),
		"DOMAIN=ADDRESS[,ADDRESS...] split by blanks, as example.internal=10.20.0.53, each " +
			"domain once and in lower case, with up to 3 IPv4 addresses"},
	{"WS_CABLE_SERVICE", func(c *Conf) *string { return &c.CableService }, check.CableService,
		"a network service as networksetup -listallnetworkservices prints it, no * first, " +
			"in quotes where it holds a blank"},
	{"WS_CABLE_PORT", func(c *Conf) *string { return &c.CablePort }, plain(check.CablePort),
		"a port of ws's, as enp5s0: up to 15 letters, digits and ._-, no - first"},
	{"WS_GIT_NAME", func(c *Conf) *string { return &c.GitName }, check.GitName,
		`a name, in quotes where it holds a blank, with no " or \`},
	{"WS_GIT_EMAIL", func(c *Conf) *string { return &c.GitEmail }, plain(check.GitEmail),
		"an email address, as ws@example.com: letters, digits and ._%+- before the @, and " +
			"a host name after it"},
	{"WS_GIT_SIGNING_PUB_KEY", func(c *Conf) *string { return &c.GitSigningPubKey }, plain(check.KeyFile), keyFile},
	{"WS_IDENTITY_PUB_KEY", func(c *Conf) *string { return &c.IdentityPubKey }, plain(check.KeyFile), keyFile},
	{"WS_IDENTITY_AGENT", func(c *Conf) *string { return &c.IdentityAgent }, check.Socket,
		"an agent's socket: a path from / or ~/, of letters, digits, blanks and ._/@+-, " +
			"in quotes where it holds a blank"},
}

// isLAN reports whether v is a network clear of the cable's and the slots' subnets.
func isLAN(v string) bool {
	p, ok := check.CIDR(v)
	return ok && !p.Overlaps(tunnel.CableNet) && !p.Overlaps(slot.Range)
}
