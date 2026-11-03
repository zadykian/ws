package wireguard

import (
	"crypto/rand"
	"io"
	"time"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

// Step names. The tunnel and the key wait on the packages, which bring systemd and its group.
const (
	settingsStep = "WireGuard settings"
	packagesStep = "WireGuard packages"
	dropsStep    = "WireGuard drops"
	tunnelStep   = "WireGuard tunnel"
	keyStep      = "WireGuard key"
)

// Steps returns the role's steps in order: the check of the settings, the packages, the drops that
// keep Docker's containers off the Mac, wg0's key and files, and the public key with the wait for
// wg0. The role takes no secrets.
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, out effect.Out) []step.Step {
	return steps(v, m, effect.OSUnits{}, out, rand.Reader, time.Sleep)
}

// steps is Steps with the units, the source of random bytes and the pause given, for a test to
// fake.
func steps(v values.Values, m effect.Machine, u effect.Units, out effect.Out, random io.Reader,
	sleep func(time.Duration)) []step.Step {
	return []step.Step{
		host.Group{Title: settingsStep, Parts: []host.Resource{settingsCheck{v.Wireguard}}},
		// networkd, its directory /etc/systemd/network and its group systemd-network come with
		// systemd, which a container may lack, so that --only wireguard stands alone. iptables
		// holds the containers' drops. wireguard-tools' recommends serve wg-quick, which the role
		// doesn't use.
		host.Group{Title: packagesStep, After: settingsStep, Parts: []host.Resource{
			host.Packages{M: m, Out: out, Names: []string{"wireguard-tools", "systemd", "iptables"}},
		}},
		// Ahead of wg0, which the drops don't need: a run that stops on wg0 still puts them in.
		host.Lazy{Title: dropsStep, After: packagesStep, Parts: drops(m, u, out, v)},
		host.Lazy{Title: tunnelStep, After: dropsStep, Parts: tunnel(m, u, out, random, v)},
		host.Group{Title: keyStep, After: tunnelStep, Parts: []host.Resource{
			ready{m: m, out: out, tries: 10, pause: time.Second, sleep: sleep},
		}},
	}
}
