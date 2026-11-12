package network

import (
	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

// Steps returns the role's one step: the direct link to the Mac. The role takes no secrets.
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, u effect.Units, out effect.Out) []step.Step {
	return steps(v, m, u, out)
}

// steps is Steps with the units given, for a test to fake.
func steps(v values.Values, m effect.Machine, u effect.Units, out effect.Out) []step.Step {
	return []step.Step{host.Group{Title: "Direct link", Parts: []host.Resource{
		direct{M: m, U: u, Out: out, Interface: v.Network.DirectLinkInterface, Address: v.Network.DirectLinkAddress},
	}}}
}
