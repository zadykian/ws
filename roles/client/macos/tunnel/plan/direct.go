package plan

// Direct is a way to ws outside the tunnel.
type Direct struct{ Host, Port string }

// DirectWay picks the way to ws outside the tunnel, in order. The cable comes first, where the Mac
// has an address in its subnet. The LAN is next, where it has one in WS_LAN; then WS_REMOTE on
// remotePort, kept as a name and never resolved. It reports false where none holds.
func DirectWay(ifaces []Iface, c Conf, remotePort string) (Direct, bool) {
	switch {
	case within(ifaces, CableNet):
		return Direct{CableServer.String(), "22"}, true
	case c.LAN.IsValid() && c.LANServer.IsValid() && within(ifaces, c.LAN):
		return Direct{c.LANServer.String(), "22"}, true
	case c.Remote != "" && remotePort != "":
		return Direct{c.Remote, remotePort}, true
	}
	return Direct{}, false
}
