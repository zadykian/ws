package jetbrains

import (
	"context"
	"embed"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

//go:embed files
var files embed.FS

// Unit files and the timer's name.
const (
	timerName   = "jetbrains-update.timer"
	serviceFile = "/etc/systemd/system/jetbrains-update.service"
	timerFile   = "/etc/systemd/system/" + timerName
	// binaryDir holds root's copy of ws, which the service runs. The user may replace the ws that
	// wrote it, and what root runs must be writable by root alone.
	binaryDir  = "/usr/local/lib/ws"
	binaryFile = binaryDir + "/jetbrains"
)

// timer makes the parts that set up the weekly update: jetbrains-update.timer runs the role once a
// week, so that a new release reaches the server between runs of ws, which runs by hand. The
// service runs ws server apply --only jetbrains with a copy of the ws at exe, which is root's.
// Without systemd booted the unit files are written and the timer is left alone.
func timer(m effect.Machine, u effect.Units, out effect.Out, exe string) func(context.Context) ([]host.Resource, error) {
	return func(context.Context) ([]host.Resource, error) {
		service, err := host.Render(files, "files/jetbrains-update.service.tmpl", struct{ Exe string }{binaryFile})
		if err != nil {
			return nil, err
		}
		unit, err := files.ReadFile("files/jetbrains-update.timer")
		if err != nil {
			return nil, err
		}
		self, err := m.ReadFile(exe)
		if err != nil {
			return nil, err
		}
		serviceF := host.File{M: m, Path: serviceFile, Content: service, Mode: 0o644}
		timerF := host.File{M: m, Path: timerFile, Content: unit, Mode: 0o644}
		return []host.Resource{
			host.Dir{M: m, Path: binaryDir, Mode: 0o755},
			host.File{M: m, Path: binaryFile, Content: self, Mode: 0o755, Secret: true},
			serviceF, timerF,
			host.DaemonReload{M: m, U: u, Watch: []host.Resource{serviceF, timerF}},
			host.Unit{M: m, U: u, Out: out, Name: timerName, Enabled: true, Run: host.Running},
		}, nil
	}
}
