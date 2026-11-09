package tmux

import (
	"context"
	"embed"
	"os"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

//go:embed files
var files embed.FS

const (
	barBinary = "/usr/local/lib/ws-bar"
	barUnit   = "/etc/systemd/system/ws-bar.service"
)

// barParts returns the parts of ws-bar, which writes the host's load for tmux's status lines. The
// unit runs a copy of ws, so that an update of ws leaves the running service unchanged until the
// next run. A change to the copy or the unit restarts the service. Without systemd, as in CI's
// container, no service runs, so the step skips whole. exe returns the path of the ws that runs.
func barParts(m effect.Machine, u effect.Units, out effect.Out, exe func() (string, error)) func(context.Context) ([]host.Resource, error) {
	return func(context.Context) ([]host.Resource, error) {
		if !host.Booted(m) {
			out.Say("ws-bar: skipped, systemd is not running")
			return nil, nil
		}
		path, err := exe()
		if err != nil {
			return nil, err
		}
		self, err := m.ReadFile(path)
		if err != nil {
			return nil, err
		}
		unit, err := files.ReadFile("files/ws-bar.service")
		if err != nil {
			return nil, err
		}
		binary := host.File{M: m, Path: barBinary, Content: self, Mode: 0o755, Secret: true}
		unitFile := host.File{M: m, Path: barUnit, Content: unit, Mode: 0o644}
		return []host.Resource{binary, unitFile,
			host.DaemonReload{M: m, U: u, Watch: []host.Resource{unitFile}},
			host.Unit{M: m, U: u, Out: out, Name: "ws-bar.service", Enabled: true, Run: host.Running, Watch: []host.Resource{binary, unitFile}}}, nil
	}
}

// running is the path of the ws that runs.
func running() (string, error) { return os.Executable() }
