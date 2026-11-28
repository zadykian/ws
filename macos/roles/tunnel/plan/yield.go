package plan

import (
	"fmt"
	"net/netip"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/check"
)

// ParseYield reads the networks that ws macos tunnel yield gives to another owner, a network a
// line, each once and in the file's order. A daemon refuses no yield: it skips a line that is no
// IPv4 network, with a warning.
func ParseYield(text string) (nets []netip.Prefix, warnings []string) {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		p, ok := check.CIDR(line)
		if !ok {
			warnings = append(warnings, fmt.Sprintf("the yield's line '%s' is no IPv4 network; skips it", line))
			continue
		}
		if !slices.Contains(nets, p) {
			nets = append(nets, p)
		}
	}
	return nets, warnings
}
