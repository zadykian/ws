// Package roles lists the server's roles in the order of site.yml. A role is either ported, with
// Go steps that its package's Steps returns, or left to the playbook, which runs it by its tag.
package roles

import (
	"slices"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/server/base"
	"github.com/zadykian/ws/internal/server/devtools"
	"github.com/zadykian/ws/internal/server/docker"
	"github.com/zadykian/ws/internal/server/ikeward"
	"github.com/zadykian/ws/internal/server/jetbrains"
	"github.com/zadykian/ws/internal/server/network"
	"github.com/zadykian/ws/internal/server/otelcol"
	"github.com/zadykian/ws/internal/server/secrets"
	"github.com/zadykian/ws/internal/server/security"
	"github.com/zadykian/ws/internal/server/shell"
	"github.com/zadykian/ws/internal/server/signoz"
	"github.com/zadykian/ws/internal/server/tmux"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/internal/server/wireguard"
	"github.com/zadykian/ws/step"
)

// Role is one role of the server.
type Role struct {
	// Name is the role's name, and the tag of its roles/NAME in site.yml.
	Name string
	// Steps returns the role's steps, in the order they run. It is nil where the playbook runs
	// the role. A ported role's package gives it as `Steps: name.Steps`, with Steps declared as
	// func Steps(v values.Values, s secrets.Secrets, m effect.Machine, out effect.Out) []step.Step.
	Steps func(v values.Values, s secrets.Secrets, m effect.Machine, out effect.Out) []step.Step
}

// Ported reports whether the role runs as Go steps.
func (r Role) Ported() bool { return r.Steps != nil }

// all is every role, in the order of site.yml. A port sets its role's Steps and removes the role
// from site.yml, which TestPlaybookRolesAreSiteYml checks.
var all = []Role{
	{Name: "base", Steps: base.Steps},
	{Name: "security", Steps: security.Steps},
	{Name: "network", Steps: network.Steps},
	{Name: "wireguard", Steps: wireguard.Steps},
	{Name: "ikeward", Steps: ikeward.Steps},
	{Name: "fans"},
	{Name: "shell", Steps: shell.Steps},
	{Name: "tmux", Steps: tmux.Steps},
	{Name: "devtools", Steps: devtools.Steps},
	{Name: "docker", Steps: docker.Steps},
	{Name: "claude"},
	{Name: "signoz", Steps: signoz.Steps},
	{Name: "otelcol", Steps: otelcol.Steps},
	{Name: "helix"},
	{Name: "jetbrains", Steps: jetbrains.Steps},
}

// List returns every role, in the order that they run.
func List() []Role { return slices.Clone(all) }

// Playbook returns the names of the roles that the playbook runs, in site.yml's order.
func Playbook() []string {
	var names []string
	for _, r := range all {
		if !r.Ported() {
			names = append(names, r.Name)
		}
	}
	return names
}
