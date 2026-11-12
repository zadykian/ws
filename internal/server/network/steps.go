package network

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
)

// Steps returns the role's one step: the direct link to the Mac. The role takes no secrets.
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, u effect.Units, out effect.Out) []step.Step {
	return []step.Step{host.Group{Title: "Direct link", Parts: []host.Resource{
		direct{M: m, U: u, Out: out, Interface: v.Network.DirectLinkInterface, Address: v.Network.DirectLinkAddress},
	}}}
}
