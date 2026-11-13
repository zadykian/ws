package wireguard

import (
	"context"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
)

// drops are the resources of the script and the unit that keep Docker's containers off the Mac.
// Through wg0 or the cable they come from the server's address, and the Mac lets them in as the
// server's. The role runs before docker's, so a first run has the drops in place as Docker first
// starts. Where Docker runs already, the restart puts them in force at once. Docker's daemon.json
// leaves it on its iptables backend, which docker-user.sh needs. WantedBy=docker.service: Docker's
// start, at boot too, starts the unit first.
func drops(m effect.Machine, u effect.Units, out effect.Out, cable string) func(context.Context) ([]host.Resource, error) {
	return func(context.Context) ([]host.Resource, error) {
		return dropResources(m, u, out, cable)
	}
}

func dropResources(m effect.Machine, u effect.Units, out effect.Out, cable string) ([]host.Resource, error) {
	script, err := DropsScript()
	if err != nil {
		return nil, err
	}
	service, err := DropsService(MacInterfaces(cable))
	if err != nil {
		return nil, err
	}
	scriptFile := host.File{M: m, Path: dropsScript, Content: script, Mode: 0o755}
	serviceFile := host.File{M: m, Path: dropsUnit, Content: service, Mode: 0o644}
	return []host.Resource{
		host.Dir{M: m, Path: "/usr/local/lib/ws", Mode: 0o755},
		host.Dir{M: m, Path: dropsDir, Mode: 0o755},
		scriptFile, serviceFile,
		host.DaemonReload{M: m, U: u, Watch: []host.Resource{serviceFile}},
		host.Unit{M: m, U: u, Out: out, Name: "wireguard-docker-user.service", Enabled: true, Run: host.Running,
			Watch: []host.Resource{scriptFile, serviceFile}},
	}, nil
}
