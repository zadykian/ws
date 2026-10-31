package conf

import (
	"net/netip"
	"strings"

	"github.com/zadykian/ws/mac/internal/check"
)

// pairs checks the keys that go together, as Parse read them, and returns the first refusal. Each
// key passed its own check, so that a network and an address parse.
func pairs(c Conf) []Error {
	refuse := func(msg string) []Error { return []Error{{0, msg}} }
	switch {
	case (c.LAN == "") != (c.LANAddress == ""):
		return refuse("WS_LAN and WS_LAN_ADDRESS go together")
	case c.LAN != "" && !check.LANHost(netip.MustParsePrefix(c.LAN), netip.MustParseAddr(c.LANAddress)):
		form, _ := lookup("WS_LAN_ADDRESS")
		return refuse("WS_LAN_ADDRESS=" + c.LANAddress + " is not " + form.form + ", " + c.LAN)
	case c.RemotePort != "" && c.Remote == "":
		return refuse("WS_REMOTE_PORT needs WS_REMOTE")
	// A cloud VM is reached at its name or address alone, and a VM on the Mac on its network alone.
	case c.Target == "cloud" && c.RemotePort == "":
		return refuse("WS_TARGET=cloud needs WS_REMOTE and WS_REMOTE_PORT")
	case c.Target == "mac-vm" && c.LAN == "":
		return refuse("WS_TARGET=mac-vm needs WS_LAN")
	}
	if away := foreign(c); len(away) > 0 {
		return refuse("WS_TARGET=" + c.Target + " has another target's key: " +
			strings.Join(away, " ") + " (docs/mac-targets.md)")
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
	if c.Target == "mac-vm" && c.Remote != "" {
		away = append(away, "WS_REMOTE")
	}
	if c.Target != "lan" && c.CableService != "" {
		away = append(away, "WS_CABLE_SERVICE")
	}
	if c.Target != "lan" && c.CablePort != "" {
		away = append(away, "WS_CABLE_PORT")
	}
	return away
}
