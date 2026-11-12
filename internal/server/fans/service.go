package fans

import (
	"context"
	"os"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

const (
	binaryPath = "/usr/local/lib/ws-fans"
	unitPath   = "/etc/systemd/system/ws-fans.service"
)

// serviceParts returns the parts of ws-fans, which drives the board's fans on the curves of its
// config, checked by the daemon's own loader. The unit runs a copy of ws, so that an update of ws
// leaves the running service unchanged until the next run. A change to the copy, the config or the
// unit restarts the service, so the three are one step. The service hands the fans back as it
// stops, and takes them again from their speeds as it starts. exe returns the path of the ws that
// runs.
func serviceParts(m effect.Machine, u effect.Units, out effect.Out, f values.Fans, exe func() (string, error)) func(context.Context) ([]host.Resource, error) {
	return func(context.Context) ([]host.Resource, error) {
		config, err := Config(f)
		if err != nil {
			return nil, err
		}
		path, err := exe()
		if err != nil {
			return nil, err
		}
		self, err := m.ReadFile(path)
		if err != nil {
			return nil, err
		}
		unit, err := files.ReadFile("files/ws-fans.service")
		if err != nil {
			return nil, err
		}
		binary := host.File{M: m, Path: binaryPath, Content: self, Mode: 0o755, Secret: true}
		conf := host.File{M: m, Path: configPath, Content: config, Mode: 0o644}
		unitFile := host.File{M: m, Path: unitPath, Content: unit, Mode: 0o644}
		watch := []host.Resource{binary, conf, unitFile}
		return append(watch,
			host.DaemonReload{M: m, U: u, Watch: []host.Resource{unitFile}},
			host.Unit{M: m, U: u, Out: out, Name: "ws-fans.service", Enabled: true, Run: host.Running, Watch: watch}), nil
	}
}

// running is the path of the ws that runs.
func running() (string, error) { return os.Executable() }
