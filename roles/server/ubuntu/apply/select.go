package apply

import (
	"fmt"
	"slices"
	"strings"
)

// Select returns the roles that only names, each by ROLE or ROLE/STEP, or every role of list where
// only is empty, in the order of list. Config.CheckSteps checks the steps.
func Select(list []Role, only []string) ([]Role, error) {
	named := make([]string, len(only))
	for i, entry := range only {
		named[i], _, _ = strings.Cut(entry, "/")
		if !slices.ContainsFunc(list, func(r Role) bool { return r.Name == named[i] }) {
			names := make([]string, len(list))
			for i, r := range list {
				names[i] = r.Name
			}
			return nil, fmt.Errorf("no role %q; the roles are %s", named[i], strings.Join(names, ", "))
		}
	}
	if len(only) == 0 {
		return list, nil
	}
	return slices.DeleteFunc(slices.Clone(list), func(r Role) bool { return !slices.Contains(named, r.Name) }), nil
}

// CheckSteps refuses a ROLE/STEP of Only that names no step of its role. It comes past Select, as
// a role's steps follow the machine's values: ikeward has none without a profile.
func (c Config) CheckSteps(list []Role) error {
	for _, r := range list {
		for _, entry := range c.Only {
			name, ok := strings.CutPrefix(entry, r.Name+"/")
			if !ok {
				continue
			}
			var names []string
			for _, s := range c.Steps(r) {
				names = append(names, s.Name())
			}
			if !slices.Contains(names, name) {
				return fmt.Errorf("no step %q in %s; its steps here are %q", name, r.Name, names)
			}
		}
	}
	return nil
}

// leftOut is the filter of a run's roles: a role named by ROLE/STEP alone keeps those steps. With
// no Only it returns nil, as every role runs whole.
func (o Options) leftOut() func(role, step string) bool {
	if len(o.Only) == 0 {
		return nil
	}
	return func(role, step string) bool {
		return !slices.Contains(o.Only, role) && !slices.Contains(o.Only, role+"/"+step)
	}
}
