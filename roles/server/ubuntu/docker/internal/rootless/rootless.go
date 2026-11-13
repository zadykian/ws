// Package rootless sets up ws_user's own Docker daemon: its subordinate ids, its drop-ins and the
// tool that installs its unit.
package rootless

import (
	"fmt"
	"path"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
)

// Resources makes the resources of ws_user's own daemon from its daemon.json and
// docker.slice, which the role works out from the values.
func Resources(m effect.Machine, u effect.Units, out effect.Out, a host.Account,
	daemonJSON, slice []byte) ([]host.Resource, error) {
	override, err := overrideDropIn()
	if err != nil {
		return nil, err
	}
	delegate, err := delegateDropIn()
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
		subIDs{m, a.Name},
		host.Dir{M: m, Path: path.Dir(delegatePath), Mode: 0o755},
		delegation,
		host.DaemonReload{M: m, U: u, Watch: []host.Resource{delegation}},
		host.Linger{M: m, User: a.User, Out: out},
		host.Unit{M: m, U: u, Out: out, Name: fmt.Sprintf("user@%d.service", a.UID), Run: host.Running},
		daemon,
		sliceFile,
		service,
		rootlessInstall{m, a, out},
		host.UserUnit{M: m, Account: a, Out: out, Name: "docker.service",
			Watch: []host.Resource{daemon, sliceFile, service}},
	}, nil
}
