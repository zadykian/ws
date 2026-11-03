// Package security is the security role as Go steps: sshd by key only, ufw, fail2ban and nft's
// outbound table. See docs/security.md.
package security

import (
	"context"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/server/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

// Step names. Those after the packages wait on them: a check run says "after Security packages"
// for them, as a machine without the packages cannot show their plans.
const (
	packagesStep = "Security packages"
	keysStep     = "SSH keys"
	sshdStep     = "sshd"
	ufwStep      = "ufw"
	fail2banStep = "fail2ban"
	outboundStep = "Outbound table"
	offStep      = "Outbound table off"
)

// Steps returns the role's steps in order: the packages, the check of the user's keys, sshd, ufw,
// fail2ban, and the outbound table, or its removal where the value is off. The role takes no secrets.
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, out effect.Out) []step.Step {
	return steps(v, m, effect.OSUnits{}, out)
}

// steps is Steps with the units given, for a test to fake.
func steps(v values.Values, m effect.Machine, u effect.Units, out effect.Out) []step.Step {
	ss := []step.Step{
		// nftables brings nft, which loads the outbound table. Its own service stays off, as the
		// package leaves it. Recommends: apt's default, as the role installed them.
		host.Group{Title: packagesStep, Parts: []host.Resource{
			host.Packages{M: m, Out: out, Recommends: true,
				Names: []string{"openssh-server", "ufw", "fail2ban", "nftables"}},
		}},
		host.Group{Title: keysStep, Parts: []host.Resource{keyCheck{m, v.Home}}},
		host.Lazy{Title: sshdStep, After: packagesStep, Parts: func(context.Context) ([]host.Resource, error) {
			return sshd(m, u, out)
		}},
		host.Group{Title: ufwStep, After: packagesStep, Parts: firewall(m, u, out)},
		host.Lazy{Title: fail2banStep, After: packagesStep, Parts: func(context.Context) ([]host.Resource, error) {
			return fail2ban(m, u, out)
		}},
	}
	if !v.Security.Outbound {
		return append(ss, host.Group{Title: offStep, Parts: outboundOff(m, u, out)})
	}
	return append(ss, host.Lazy{Title: outboundStep, After: packagesStep, Parts: func(context.Context) ([]host.Resource, error) {
		return outbound(m, u, out)
	}})
}
