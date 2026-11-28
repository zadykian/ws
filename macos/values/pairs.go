package values

import (
	"net/netip"
	"strings"

	"github.com/zadykian/ws/engine/core/check"
)

// pairs checks the keys that go together, as Parse read them, and returns the first refusal. Each
// key passed its own check, so that a network and an address parse.
func pairs(c Conf) []Error {
	if c.Host != "" {
		return guestPairs(c)
	}
	refuse := func(msg string) []Error { return []Error{{0, msg}} }
	switch {
	case (c.LAN == "") != (c.LANAddress == ""):
		return refuse("WS_LAN and WS_LAN_ADDRESS go together")
	case c.LAN != "" && !check.LANHost(netip.MustParsePrefix(c.LAN), netip.MustParseAddr(c.LANAddress)):
		form, _ := lookup("WS_LAN_ADDRESS")
		return refuse("WS_LAN_ADDRESS=" + c.LANAddress + " is not " + form.form + ", " + c.LAN)
	case c.RemotePort != "" && c.Remote == "":
		return refuse("WS_REMOTE_PORT needs WS_REMOTE")
	// A cloud VM is reached at its name or address alone.
	case c.Target == "cloud" && c.RemotePort == "":
		return refuse("WS_TARGET=cloud needs WS_REMOTE and WS_REMOTE_PORT")
	}
	if away := foreign(c); len(away) > 0 {
		return refuse("WS_TARGET=" + c.Target + " has another target's key: " +
			strings.Join(away, " ") + " (macos/values/README.md)")
	}
	return nil
}

// guestPairs checks a guest's file, whose WS_HOST gives the way: the host's endpoint, its router's
// name and its cable. The guest has none of its own, and no port that a router forwards to its 22.
func guestPairs(c Conf) []Error {
	var own []string
	for _, k := range []struct{ name, value string }{
		{"WS_LAN", c.LAN}, {"WS_LAN_ADDRESS", c.LANAddress}, {"WS_REMOTE", c.Remote},
		{"WS_REMOTE_PORT", c.RemotePort}, {"WS_CABLE_SERVICE", c.CableService}, {"WS_CABLE_PORT", c.CablePort},
	} {
		if k.value != "" {
			own = append(own, k.name)
		}
	}
	if c.Target != "lan" {
		own = append(own, "WS_TARGET")
	}
	if len(own) > 0 {
		return []Error{{0, "WS_HOST=" + c.Host + " names the host's way: a guest holds none of " +
			strings.Join(own, " ") + " (macos/values/README.md)"}}
	}
	return nil
}

// foreign lists the keys of another target's way, which would lead ssh and the tunnel to a host
// that this target lacks.
func foreign(c Conf) []string {
	var away []string
	if c.Target == "cloud" && c.LAN != "" {
		away = append(away, "WS_LAN")
	}
	if c.Target != "lan" && c.CableService != "" {
		away = append(away, "WS_CABLE_SERVICE")
	}
	if c.Target != "lan" && c.CablePort != "" {
		away = append(away, "WS_CABLE_PORT")
	}
	return away
}
