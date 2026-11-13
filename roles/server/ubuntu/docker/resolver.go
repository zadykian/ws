package docker

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/server/ubuntu/internal/ufw"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// resolverComment marks the ufw rules that let Docker's networks reach the resolver.
const resolverComment = "DNS from containers"

// resolverBridges are the interfaces of Docker's bridges: docker0, and br-+ for the rest.
var resolverBridges = []string{"docker0", "br-+"}

// resolver makes the ufw rules that let Docker's networks reach the resolver on the default bridge.
func resolver(m effect.Machine, out effect.Out, d values.Docker) ufw.Rules {
	return ufw.Rules{M: m, Out: out, Comment: resolverComment, Want: resolverRules(d), Owns: ownsResolverRule, NeedsSystemd: true}
}

// resolverRules allows lookups from Docker's bridges to the resolver on the default bridge.
func resolverRules(d values.Docker) []ufw.Rule {
	var rules []ufw.Rule
	for _, b := range resolverBridges {
		rules = append(rules, ufw.Rule{Action: "allow", Proto: "any", To: d.BridgeIP, ToPort: "53",
			From: ufw.Any, FromPort: "any", Dir: "in", Interface: b})
	}
	return rules
}

// ownsResolverRule tells the resolver's rules from others that carry its comment: a rule that
// allows any protocol, from any port, to a port, in on an interface. One of another form, such as
// one that names a protocol or no interface, is left alone.
func ownsResolverRule(r ufw.Rule) bool {
	return r.Action == "allow" && r.Proto == "any" && r.FromPort == "any" && r.ToPort != "any" &&
		r.Dir == "in" && r.Interface != ""
}
