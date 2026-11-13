// Package docker is the docker role as Go steps. They set up Docker Engine from Docker's apt
// repository, its daemon.json, docker.slice, the build cache's collection and the containers' DNS
// through the host. For a ws_user other than root, they set up a rootless daemon of its own. See
// docs/docker.md.
package docker

import (
	"net/netip"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/apt"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/docker/internal/bridge"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
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
	rootlessStep   = "Docker rootless daemon"
)

// Steps returns the role's steps in order. The repository and the packages come first, then the
// check of the routes and the user's leaving the docker group. Then come the configuration, the
// check of docker0, and the user's rootless daemon. The role takes no secrets.
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, u effect.Units, out effect.Out) []step.Step {
	return steps(v, m, u, out, StatfsSize, bridge.Addrs)
}

// steps is Steps with the disk and docker0 given, for a test to fake.
func steps(v values.Values, m effect.Machine, u effect.Units, out effect.Out, size DiskSize,
	held func() ([]netip.Prefix, error)) []step.Step {
	d := v.Docker
	steps := []step.Step{
		resource.Lazy{Title: repositoryStep, Parts: repository(m)},
		// iproute2, from Ubuntu's archive, gives the route check its ip; a container lacks it.
		resource.Group{Title: packagesStep, After: repositoryStep, Parts: []resource.Resource{
			apt.Packages{M: m, Out: out, Recommends: true, Names: packages(v)},
		}},
		resource.Group{Title: routesStep, After: packagesStep, Parts: []resource.Resource{bridge.Routes{M: m, D: d, Bridge: bridgeCIDR(d)}}},
	}
	if v.User != "root" {
		steps = append(steps, resource.Group{Title: membersStep, After: packagesStep, Parts: []resource.Resource{resource.Left{M: m, User: v.User, Group: "docker"}}})
	}
	steps = append(steps,
		resource.Lazy{Title: configStep, After: routesStep, Parts: configuration(m, u, out, d, size)},
		resource.Group{Title: bridgeStep, After: configStep, Parts: []resource.Resource{
			bridge.Check{M: m, Out: out, Held: held, Want: bridgeCIDR(d)},
		}},
	)
	if v.User != "root" {
		steps = append(steps, rootlessDaemon(v, m, u, out, size))
	}
	return steps
}
