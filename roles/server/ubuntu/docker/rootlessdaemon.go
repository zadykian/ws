package docker

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// rootlessDaemon sets up ws_user's own daemon. Root writes the user's files, whose links it refuses.
// The user runs Docker's tool, which needs the user's systemd up. So lingering comes first, and the
// files, as the tool starts the daemon at once. In a container the files are written and the rest
// skips: the job that proves the rest runs on a booted machine.
func rootlessDaemon(v values.Values, m effect.Machine, u effect.Units, out effect.Out, size DiskSize) step.Step {
	user := host.User{Name: v.User, Home: v.Home}
	return user.WithAccount(m, rootlessStep, configStep, func(a host.Account) ([]host.Resource, error) {
		d, err := WithLimits(m, v.Docker)
		if err != nil {
			return nil, err
		}
		cache, err := cacheLimits(m, d, size)
		if err != nil {
			return nil, err
		}
		daemonJSON, err := RootlessJSON(d, cache)
		if err != nil {
			return nil, err
		}
		slice, err := Slice(d)
		if err != nil {
			return nil, err
		}
		return rootlessResources(m, u, out, a, daemonJSON, slice)
	})
}
