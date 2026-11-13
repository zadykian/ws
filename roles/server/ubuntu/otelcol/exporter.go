package otelcol

import (
	"context"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/host/apt"
	"github.com/zadykian/ws/engine/core/host/systemd"
)

// exporter makes the parts that set up node_exporter, which gives the temperatures and fans that
// the collector's host metrics lack. The arguments go in before the package, whose postinst starts
// node_exporter with them: with the package's own, it would serve every collector on every
// address. The file is one of the package's conffiles, and apt keeps the one it finds. The package
// comes without its recommends, which add textfile scripts for collectors it leaves off. A change
// of the arguments restarts node_exporter here rather than at the collector's step, as a run cut
// short between the two would lose the restart.
func exporter(m effect.Machine, u effect.Units, out effect.Out) func(context.Context) ([]host.Resource, error) {
	return func(context.Context) ([]host.Resource, error) {
		args, err := files.ReadFile("files/prometheus-node-exporter")
		if err != nil {
			return nil, err
		}
		file := host.File{M: m, Path: "/etc/default/prometheus-node-exporter", Content: args, Mode: 0o644}
		return []host.Resource{
			file,
			apt.Packages{M: m, Out: out, Names: []string{"prometheus-node-exporter"}},
			systemd.Unit{M: m, U: u, Out: out, Name: "prometheus-node-exporter.service", Enabled: true, Run: systemd.Running, Watch: []host.Resource{file}},
		}, nil
	}
}
