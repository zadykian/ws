package tunnel

import (
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

// TierNone is the state's tier where no path to ws answers.
const TierNone Tier = "none"

// State is what the daemon writes for status: the way in use and the routes.
type State struct {
	Tier      Tier
	Endpoint  string // host:port, or empty with TierNone
	Interface string
	Since     time.Time // when the way last changed
	Routes    []netip.Prefix
	LeftOut   []netip.Prefix // the networks of routes.d that hold the endpoint
}

// Encode is the state file's text: a key=value line each, the networks joined by spaces.
func (s State) Encode() string {
	return fmt.Sprintf("tier=%s\nendpoint=%s\ninterface=%s\nsince=%d\nroutes=%s\nleft_out=%s\n",
		s.Tier, s.Endpoint, s.Interface, s.Since.Unix(), joinNets(s.Routes), joinNets(s.LeftOut))
}

// ParseState reads what Encode wrote. It leaves out a line it does not know.
func ParseState(text string) (s State, err error) {
	for _, line := range strings.Split(text, "\n") {
		name, value, _ := strings.Cut(line, "=")
		switch name {
		case "tier":
			s.Tier = Tier(value)
		case "endpoint":
			s.Endpoint = value
		case "interface":
			s.Interface = value
		case "since":
			n, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return State{}, fmt.Errorf("state: since is '%s'", value)
			}
			s.Since = time.Unix(n, 0)
		case "routes", "left_out":
			var nets []netip.Prefix
			for _, each := range strings.Fields(value) {
				p, err := netip.ParsePrefix(each)
				if err != nil {
					return State{}, fmt.Errorf("state: %s holds '%s'", name, each)
				}
				nets = append(nets, p)
			}
			if name == "routes" {
				s.Routes = nets
			} else {
				s.LeftOut = nets
			}
		}
	}
	return s, nil
}

func joinNets(nets []netip.Prefix) string {
	parts := make([]string, len(nets))
	for i, each := range nets {
		parts[i] = each.String()
	}
	return strings.Join(parts, " ")
}
