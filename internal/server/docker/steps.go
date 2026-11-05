// Package docker is the docker role as Go steps: Docker Engine from Docker's apt repository, its
// daemon.json, docker.slice, the build cache's collection and the containers' DNS through the
// host. See docs/docker.md.
package docker

import (
	"net/netip"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

// Step names. Those after the packages wait on them: a check run says "after Docker packages" for
// them, as a machine without Docker's packages cannot show their plans.
const (
	repositoryStep = "Docker repository"
	packagesStep   = "Docker packages"
	routesStep     = "Docker routes"
	membersStep    = "Docker group"
	configStep     = "Docker configuration"
	bridgeStep     = "Docker bridge"
)

// Steps returns the role's steps in order: the repository, the packages, the check of the routes,
// the user's group, the configuration, and the check of docker0. The role takes no secrets.
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, out effect.Out) []step.Step {
	return steps(v, m, effect.OSUnits{}, out, StatfsSize, BridgeAddrs)
}

// steps is Steps with the units, the disk and docker0 given, for a test to fake.
func steps(v values.Values, m effect.Machine, u effect.Units, out effect.Out, size DiskSize,
	held func() ([]netip.Prefix, error)) []step.Step {
	d := v.Docker
	steps := []step.Step{
		host.Lazy{Title: repositoryStep, Parts: repository(m)},
		// iproute2, from Ubuntu's archive, gives the route check its ip; a container lacks it.
		host.Group{Title: packagesStep, After: repositoryStep, Parts: []host.Resource{
			host.Packages{M: m, Out: out, Recommends: true, Names: packages(v)},
		}},
		host.Group{Title: routesStep, After: packagesStep, Parts: []host.Resource{routeCheck{m, d}}},
	}
	if v.User != "root" {
		steps = append(steps, host.Group{Title: membersStep, After: packagesStep, Parts: []host.Resource{host.Left{M: m, User: v.User, Group: "docker"}}})
	}
	return append(steps,
		host.Lazy{Title: configStep, After: routesStep, Parts: configuration(m, u, out, d, size)},
		host.Group{Title: bridgeStep, After: configStep, Parts: []host.Resource{
			bridgeCheck{m: m, out: out, held: held, want: bridgeCIDR(d)},
		}},
	)
}
