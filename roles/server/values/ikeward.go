package values

import (
	"errors"
	"fmt"
	"net/netip"
	"path"
	"strings"
	"unicode"
)

// check refuses the values the ikeward role cannot use, where a profile is set. The profile's path
// names the files that install writes. The role's units quote it in TOML's double quotes, so check
// refuses a quote, a backslash and a control character, which would end the string or break it.
// ikeward install refuses to run without the resolver and a domain, and its error names its flags,
// not these keys. ufw prints a rule's address as given, and the removal of a peer's rule compares
// the two. Without a profile, nothing is checked, as the role is off.
func (v Ikeward) check() error {
	if v.Profile == "" {
		return nil
	}
	if strings.ContainsFunc(v.Profile, func(r rune) bool { return r == '"' || r == '\\' || unicode.IsControl(r) }) {
		return fmt.Errorf("WS_IKEWARD_PROFILE %q holds a quote, a backslash or a control character, "+
			"which no profile's path takes", v.Profile)
	}
	if !path.IsAbs(v.Profile) {
		return fmt.Errorf("WS_IKEWARD_PROFILE is %s, which is not an absolute path", v.Profile)
	}
	if v.DNS == "" {
		return errors.New("a profile needs WS_IKEWARD_DNS, the remote network's resolver")
	}
	if len(v.Domains) == 0 {
		return errors.New("a profile needs at least one domain in WS_IKEWARD_DOMAINS")
	}
	var bad []string
	for _, p := range v.ForwardFrom {
		if a, err := netip.ParseAddr(p); err != nil || !a.Is4() {
			bad = append(bad, p)
		}
	}
	if len(bad) > 0 {
		return fmt.Errorf("WS_IKEWARD_FORWARD_FROM holds %s: each peer is an IPv4 address, as 10.99.0.2, "+
			"with no prefix length", strings.Join(bad, ", "))
	}
	return nil
}
