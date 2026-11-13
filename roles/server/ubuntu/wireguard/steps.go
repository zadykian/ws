// Package wireguard is the wireguard role as Go steps. It sets up wg0, the server's end of a
// WireGuard tunnel for the Mac, from networkd's own files. It also sets up the drops that keep
// Docker's containers off the Mac. See docs/wireguard.md.
package wireguard

import (
	"crypto/rand"
	"io"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/apt"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
	"github.com/zadykian/ws/roles/server/ubuntu/wireguard/internal/drops"
	"github.com/zadykian/ws/roles/server/ubuntu/wireguard/internal/settings"
)

// Step names. The tunnel and the key wait on the packages, which bring systemd and its group.
const (
	settingsStep = "WireGuard settings"
	packagesStep = "WireGuard packages"
	dropsStep    = "WireGuard drops"
	tunnelStep   = "WireGuard tunnel"
	keyStep      = "WireGuard key"
)

// Steps returns the role's steps in order: the check of the settings, the packages, and the drops
// that keep Docker's containers off the Mac. Then come wg0's key and files, and the public key with
// the wait for wg0. The role takes no secrets.
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, u effect.Units, out effect.Out) []step.Step {
	return steps(v, m, u, out, rand.Reader, time.Sleep)
}

// steps is Steps with the source of random bytes and the pause given, for a test to fake.
func steps(v values.Values, m effect.Machine, u effect.Units, out effect.Out, random io.Reader,
	sleep func(time.Duration)) []step.Step {
	return []step.Step{
		resource.Group{Title: settingsStep, Parts: []resource.Resource{settings.Check{W: v.Wireguard}}},
		// networkd, its directory /etc/systemd/network and its group systemd-network come with
		// systemd, which a container may lack, so that --only wireguard stands alone. iptables
		// holds the containers' drops. wireguard-tools' recommends serve wg-quick, which the role
		// doesn't use.
		resource.Group{Title: packagesStep, After: settingsStep, Parts: []resource.Resource{
			apt.Packages{M: m, Out: out, Names: []string{"wireguard-tools", "systemd", "iptables"}},
		}},
		// Ahead of wg0, which the drops don't need: a run that stops on wg0 still puts them in.
		resource.Lazy{Title: dropsStep, After: packagesStep, Parts: drops.Parts(m, u, out, v.Network.DirectLinkInterface)},
		resource.Lazy{Title: tunnelStep, After: dropsStep, Parts: tunnel(m, u, out, random, v)},
		resource.Group{Title: keyStep, After: tunnelStep, Parts: []resource.Resource{
			ready{m: m, out: out, tries: 10, pause: time.Second, sleep: sleep},
		}},
	}
}
