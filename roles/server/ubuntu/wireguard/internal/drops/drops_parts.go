package drops

import (
	"context"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/resource/systemd"
)

// Parts are the resources of the script and the unit that keep Docker's containers off the Mac.
// Through wg0 or the cable they come from the server's address, and the Mac lets them in as the
// server's. The role runs before docker's, so a first run has the drops in place as Docker first
// starts. Where Docker runs already, the restart puts them in force at once. Docker's daemon.json
// leaves it on its iptables backend, which docker-user.sh needs. WantedBy=docker.service: Docker's
// start, at boot too, starts the unit first.
func Parts(m effect.Machine, u effect.Units, out effect.Out, cable string) func(context.Context) ([]resource.Resource, error) {
	return func(context.Context) ([]resource.Resource, error) {
		return dropResources(m, u, out, cable)
	}
}

func dropResources(m effect.Machine, u effect.Units, out effect.Out, cable string) ([]resource.Resource, error) {
	script, err := Script()
	if err != nil {
		return nil, err
	}
	service, err := service(MacInterfaces(cable))
	if err != nil {
		return nil, err
	}
	scriptFile := file.File{M: m, Path: dropsScript, Content: script, Mode: 0o755}
	serviceFile := file.File{M: m, Path: dropsUnit, Content: service, Mode: 0o644}
	return []resource.Resource{
		file.Dir{M: m, Path: "/usr/local/lib/ws", Mode: 0o755},
		file.Dir{M: m, Path: dropsDir, Mode: 0o755},
		scriptFile, serviceFile,
		systemd.DaemonReload{M: m, U: u, Watch: []resource.Resource{serviceFile}},
		systemd.Unit{M: m, U: u, Out: out, Name: "wireguard-docker-user.service", Enabled: true, Run: systemd.Running,
			Watch: []resource.Resource{scriptFile, serviceFile}},
	}, nil
}
