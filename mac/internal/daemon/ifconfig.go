// Package daemon runs ws tunnel's loop: it reads the Mac and the peer, asks the pure functions of
// package tunnel what to do, and does it through effect.Machine and a Device.
package daemon

import (
	"net/netip"
	"regexp"
	"slices"
	"strings"

	"github.com/zadykian/ws/mac/internal/tunnel"
)

// up matches the UP flag in the line of ifconfig that starts an interface.
var up = regexp.MustCompile(`[<,]UP[,>]`)

// ParseIfconfig lists the Mac's IPv4 addresses on interfaces that are up, the tunnel's tun aside,
// from the text of ifconfig, ordered by interface and address.
func ParseIfconfig(text, tun string) []tunnel.Iface {
	var found []tunnel.Iface
	name, isUp := "", false
	for _, line := range strings.Split(text, "\n") {
		if line != "" && line[0] != ' ' && line[0] != '\t' {
			name, _, _ = strings.Cut(line, ":")
			isUp = up.MatchString(line)
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "inet" || !isUp || name == tun {
			continue
		}
		if a, err := netip.ParseAddr(fields[1]); err == nil && a.Is4() {
			found = append(found, tunnel.Iface{Name: name, Addr: a})
		}
	}
	slices.SortFunc(found, func(a, b tunnel.Iface) int {
		return strings.Compare(a.Name+" "+a.Addr.String(), b.Name+" "+b.Addr.String())
	})
	return found
}
