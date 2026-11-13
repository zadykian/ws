package cmd

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// serverRoles closes each of list over v and s, as the runner holds neither.
func serverRoles(list []ubuntu.Role, v values.Values, s secrets.Secrets) []role.Role {
	out := make([]role.Role, len(list))
	for i, r := range list {
		out[i] = role.Role{Name: r.Name, Steps: func(m effect.Machine, u effect.Units, o effect.Out) []step.Step {
			return r.Steps(v, s, m, u, o)
		}}
	}
	return out
}
