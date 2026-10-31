package setup

import (
	"strings"
)

// The Mac's end of the cable, in the cable's /30, and that /30's mask.
const (
	cableClient = "10.77.0.2"
	cableMask   = "255.255.255.252"
)

// serviceListed reports whether networksetup -listallnetworkservices lists the service, and whether
// it lists it as disabled, which the list marks with a * first.
func serviceListed(list, service string) (listed, disabled bool) {
	for _, line := range strings.Split(list, "\n") {
		switch line {
		case service:
			return true, false
		case "*" + service:
			disabled = true
		}
	}
	return disabled, disabled
}

// cableOther returns what networksetup -getinfo shows that is not a Manual Configuration of the
// cable's address and mask with no router, joined by "; ", or nothing where the service is set.
// No source shows which lines getinfo prints while the display, and so its adapter, is away. So an
// address, mask or router line it leaves out passes, and a run docked checks the service again.
func cableOther(info string) string {
	var method, ips, masks, routers string
	first := false
	ours := false
	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimRight(line, " \t\r")
		if line == "" {
			continue
		}
		if !first {
			first = true
			if line != "Manual Configuration" {
				method = "; " + line
			}
		}
		switch {
		case line == "IP address: "+cableClient:
			ours = true
		case strings.HasPrefix(line, "IP address: "):
			ips += "; " + line
		case strings.HasPrefix(line, "Subnet mask: ") && line != "Subnet mask: "+cableMask:
			masks += "; " + line
		case len(line) > len("Router: ") && strings.HasPrefix(line, "Router: ") && line[8] >= '0' && line[8] <= '9':
			routers += "; " + line
		}
	}
	if !first {
		method = "; nothing"
	}
	if ours {
		ips = ""
	}
	return strings.TrimPrefix(method+ips+masks+routers, "; ")
}
