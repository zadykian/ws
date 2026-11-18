// Package ubuntu lists the server's roles in the order that they run. A role is the Go steps that
// its package's Steps returns.
package ubuntu

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/base"
	"github.com/zadykian/ws/roles/server/ubuntu/cable"
	"github.com/zadykian/ws/roles/server/ubuntu/claude"
	"github.com/zadykian/ws/roles/server/ubuntu/devtools"
	"github.com/zadykian/ws/roles/server/ubuntu/docker"
	"github.com/zadykian/ws/roles/server/ubuntu/git"
	"github.com/zadykian/ws/roles/server/ubuntu/ikeward"
	"github.com/zadykian/ws/roles/server/ubuntu/otelcol"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/security"
	"github.com/zadykian/ws/roles/server/ubuntu/signoz"
	"github.com/zadykian/ws/roles/server/ubuntu/tunnel"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// entry is one role of the server, not yet closed over the machine's values and secrets.
type entry struct {
	// name is the role's name, as --only takes it.
	name string
	// steps returns the role's steps, in the order they run. A role's package gives it as
	// `{"name", name.Steps}`, with Steps declared as
	// func Steps(v values.Values, s secrets.Secrets, m effect.Machine, u effect.Units, out effect.Out) []step.Step.
	steps func(v values.Values, s secrets.Secrets, m effect.Machine, u effect.Units, out effect.Out) []step.Step
}

// all is every role, in the order that they run.
var all = []entry{
	{"base", base.Steps},
	{"security", security.Steps},
	{"cable", cable.Steps},
	{"tunnel", tunnel.Steps},
	{"ikeward", ikeward.Steps},
	{"git", git.Steps},
	{"devtools", devtools.Steps},
	{"docker", docker.Steps},
	{"claude", claude.Steps},
	{"signoz", signoz.Steps},
	{"otelcol", otelcol.Steps},
}

// Roles returns every role, in the order that they run, closed over the machine's values v and
// secrets s, as the runner holds neither. Each call makes a new slice.
func Roles(v values.Values, s secrets.Secrets) []role.Role { return closeOver(all, v, s) }

// closeOver is the seam that Roles gives the list: it closes each of list over v and s.
func closeOver(list []entry, v values.Values, s secrets.Secrets) []role.Role {
	out := make([]role.Role, len(list))
	for i, r := range list {
		out[i] = role.Role{Name: r.name, Steps: func(m effect.Machine, u effect.Units, o effect.Out) []step.Step {
			return r.steps(v, s, m, u, o)
		}}
	}
	return out
}

// Names returns the names of the roles, in the order that they run, where no values are at hand.
func Names() []string {
	names := make([]string, len(all))
	for i, r := range all {
		names[i] = r.name
	}
	return names
}
