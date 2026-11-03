package apply

import (
	"fmt"
	"slices"
	"strings"

	"github.com/zadykian/ws/internal/server/roles"
)

// Select returns the roles that only names, or all of list where only is empty, in the order of
// list.
func Select(list []roles.Role, only []string) ([]roles.Role, error) {
	for _, name := range only {
		if !slices.ContainsFunc(list, func(r roles.Role) bool { return r.Name == name }) {
			names := make([]string, len(list))
			for i, r := range list {
				names[i] = r.Name
			}
			return nil, fmt.Errorf("no role %q; the roles are %s", name, strings.Join(names, ", "))
		}
	}
	if len(only) == 0 {
		return list, nil
	}
	return slices.DeleteFunc(slices.Clone(list), func(r roles.Role) bool { return !slices.Contains(only, r.Name) }), nil
}
