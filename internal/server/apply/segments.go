package apply

import (
	"fmt"
	"slices"
	"strings"

	"github.com/zadykian/ws/internal/server/roles"
)

// Segment is a run of roles: one ported role, whose steps run, or the consecutive roles that the
// playbook runs in one ansible-playbook.
type Segment struct {
	// Role is the ported role, or the zero Role where Playbook is set.
	Role roles.Role
	// Playbook names the playbook's roles, in order.
	Playbook []string
}

// Segments cuts the roles that only names, or all where only is empty, in the order of list. Roles
// the playbook runs, side by side, make one segment.
func Segments(list []roles.Role, only []string) ([]Segment, error) {
	for _, name := range only {
		if !slices.ContainsFunc(list, func(r roles.Role) bool { return r.Name == name }) {
			names := make([]string, len(list))
			for i, r := range list {
				names[i] = r.Name
			}
			return nil, fmt.Errorf("no role %q; the roles are %s", name, strings.Join(names, ", "))
		}
	}
	var segs []Segment
	for _, r := range list {
		if len(only) > 0 && !slices.Contains(only, r.Name) {
			continue
		}
		if r.Ported() {
			segs = append(segs, Segment{Role: r})
		} else if n := len(segs); n > 0 && segs[n-1].Playbook != nil {
			segs[n-1].Playbook = append(segs[n-1].Playbook, r.Name)
		} else {
			segs = append(segs, Segment{Playbook: []string{r.Name}})
		}
	}
	return segs, nil
}
