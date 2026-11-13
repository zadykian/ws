package docker

import (
	"fmt"
	"path"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// rootless sets up ws_user's own daemon. Root writes the user's files, whose links it refuses. The
// user runs Docker's tool, which needs the user's systemd up. So lingering comes first, and the
// files, as the tool starts the daemon at once. In a container the files are written and the rest
// skips: the job that proves the rest runs on a booted machine.
func rootless(v values.Values, m effect.Machine, u effect.Units, out effect.Out, size DiskSize) step.Step {
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
		override, err := RootlessOverride()
		if err != nil {
			return nil, err
		}
		delegate, err := Delegate()
		if err != nil {
			return nil, err
		}
		daemon := host.Validated{
			File: a.File(m, ".config/docker/daemon.json", daemonJSON, 0o644),
			Check: func(path string) (string, []string) {
				return "dockerd", []string{"--validate", "--config-file", path}
			},
		}
		sliceFile := a.File(m, ".config/systemd/user/docker.slice", slice, 0o644)
		service := a.File(m, ".config/systemd/user/docker.service.d/ws.conf", override, 0o644)
		delegation := host.File{M: m, Path: delegatePath, Content: delegate, Mode: 0o644}
		return []host.Resource{
			subIDs{m, v.User},
			host.Dir{M: m, Path: path.Dir(delegatePath), Mode: 0o755},
			delegation,
			host.DaemonReload{M: m, U: u, Watch: []host.Resource{delegation}},
			host.Linger{M: m, User: user, Out: out},
			host.Unit{M: m, U: u, Out: out, Name: fmt.Sprintf("user@%d.service", a.UID), Run: host.Running},
			daemon,
			sliceFile,
			service,
			rootlessInstall{m, a, out},
			host.UserUnit{M: m, Account: a, Out: out, Name: "docker.service",
				Watch: []host.Resource{daemon, sliceFile, service}},
		}, nil
	})
}

const delegatePath = "/etc/systemd/system/user@.service.d/delegate.conf"
