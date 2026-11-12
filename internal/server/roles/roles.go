// Package roles lists the server's roles in the order that they run. A role is the Go steps that
// its package's Steps returns.
package roles

import (
	"slices"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/base"
	"github.com/zadykian/ws/internal/server/claude"
	"github.com/zadykian/ws/internal/server/devtools"
	"github.com/zadykian/ws/internal/server/docker"
	"github.com/zadykian/ws/internal/server/fans"
	"github.com/zadykian/ws/internal/server/fresh"
	"github.com/zadykian/ws/internal/server/ikeward"
	"github.com/zadykian/ws/internal/server/lazygit"
	"github.com/zadykian/ws/internal/server/network"
	"github.com/zadykian/ws/internal/server/nvidia"
	"github.com/zadykian/ws/internal/server/otelcol"
	"github.com/zadykian/ws/internal/server/security"
	"github.com/zadykian/ws/internal/server/shell"
	"github.com/zadykian/ws/internal/server/signoz"
	"github.com/zadykian/ws/internal/server/tmux"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/internal/server/wireguard"
	"github.com/zadykian/ws/internal/server/zed"
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
