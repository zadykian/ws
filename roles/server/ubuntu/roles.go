// Package ubuntu lists the server's roles in the order that they run. A role is the Go steps that
// its package's Steps returns.
package ubuntu

import (
	"slices"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/base"
	"github.com/zadykian/ws/roles/server/ubuntu/claude"
	"github.com/zadykian/ws/roles/server/ubuntu/devtools"
	"github.com/zadykian/ws/roles/server/ubuntu/docker"
	"github.com/zadykian/ws/roles/server/ubuntu/fans"
	"github.com/zadykian/ws/roles/server/ubuntu/fresh"
	"github.com/zadykian/ws/roles/server/ubuntu/ikeward"
	"github.com/zadykian/ws/roles/server/ubuntu/lazygit"
	"github.com/zadykian/ws/roles/server/ubuntu/network"
	"github.com/zadykian/ws/roles/server/ubuntu/nvidia"
	"github.com/zadykian/ws/roles/server/ubuntu/otelcol"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/security"
	"github.com/zadykian/ws/roles/server/ubuntu/shell"
	"github.com/zadykian/ws/roles/server/ubuntu/signoz"
	"github.com/zadykian/ws/roles/server/ubuntu/tmux"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
	"github.com/zadykian/ws/roles/server/ubuntu/wireguard"
	"github.com/zadykian/ws/roles/server/ubuntu/zed"
)

// Role is one role of the server.
type Role struct {
	// Name is the role's name, as --only takes it.
	Name string
	// Steps returns the role's steps, in the order they run. A role's package gives it as
	// `Steps: name.Steps`, with Steps declared as
	// func Steps(v values.Values, s secrets.Secrets, m effect.Machine, u effect.Units, out effect.Out) []step.Step.
	Steps func(v values.Values, s secrets.Secrets, m effect.Machine, u effect.Units, out effect.Out) []step.Step
}

// all is every role, in the order that they run.
var all = []Role{
	{Name: "base", Steps: base.Steps},
	{Name: "security", Steps: security.Steps},
	{Name: "network", Steps: network.Steps},
	{Name: "wireguard", Steps: wireguard.Steps},
	{Name: "ikeward", Steps: ikeward.Steps},
	{Name: "nvidia", Steps: nvidia.Steps},
	{Name: "fans", Steps: fans.Steps},
	{Name: "shell", Steps: shell.Steps},
	{Name: "tmux", Steps: tmux.Steps},
	{Name: "lazygit", Steps: lazygit.Steps},
	{Name: "devtools", Steps: devtools.Steps},
	{Name: "docker", Steps: docker.Steps},
	{Name: "claude", Steps: claude.Steps},
	{Name: "signoz", Steps: signoz.Steps},
	{Name: "otelcol", Steps: otelcol.Steps},
	{Name: "fresh", Steps: fresh.Steps},
	// zed is described in `docs/zed.md`.
	{Name: "zed", Steps: zed.Steps},
}

// List returns every role, in the order that they run.
func List() []Role { return slices.Clone(all) }
