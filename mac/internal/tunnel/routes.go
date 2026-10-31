package tunnel

import (
	"fmt"
	"net/netip"
	"slices"
	"strings"
)

// ParseRoutes reads the files of routes.d, a map of file name to contents, into the networks to
// route into the tunnel, each once, in the order of the file names. A line that is no network, or
// that overlaps the tunnel's or the cable's subnet, is left out with a warning.
func ParseRoutes(files map[string][]byte) (nets []netip.Prefix, warnings []string) {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		for _, line := range strings.Split(string(files[name]), "\n") {
			line = strings.Trim(line, " \t")
			if line == "" || line[0] == '#' {
				continue
			}
			p, ok := parseCIDR(line)
			if !ok || p.Overlaps(TunnelNet) || p.Overlaps(CableNet) {
				warnings = append(warnings,
					fmt.Sprintf("routes.d/%s: leaves out '%s', not a network it can route", name, line))
				continue
			}
			if !slices.Contains(nets, p) {
				nets = append(nets, p)
			}
		}
	}
	return nets, warnings
}
