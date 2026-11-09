package ikeward

import (
	"errors"
	"fmt"
	"net/netip"
	"path"
	"strings"

	"github.com/zadykian/ws/internal/server/values"
)

// check refuses the values the role cannot use, before anything changes. The profile's path names
// the files that install writes. ikeward install refuses to run without the resolver and a domain,
// and its error names its flags, not these keys. ufw prints a rule's address as given, and the
// removal of a peer's rule compares the two.
func check(v values.Ikeward) error {
	if !path.IsAbs(v.Profile) {
		return fmt.Errorf("ikeward_profile is %s, which is not an absolute path", v.Profile)
	}
	if v.DNS == "" {
		return errors.New("a profile needs ikeward_dns, the remote network's resolver")
	}
	if len(v.Domains) == 0 {
		return errors.New("a profile needs at least one domain in ikeward_domains")
	}
	var bad []string
	for _, p := range v.ForwardFrom {
		if a, err := netip.ParseAddr(p); err != nil || !a.Is4() {
			bad = append(bad, p)
		}
	}
	if len(bad) > 0 {
		return fmt.Errorf("ikeward_forward_from holds %s: each peer is an IPv4 address, as 10.99.0.2, "+
			"with no prefix length", strings.Join(bad, ", "))
	}
	return nil
}
