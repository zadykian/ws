package otelcol

import (
	"context"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// installStep is the name of the step that installs the .deb.
const installStep = "Collector install"

// pinned is the collector's .deb as a package, which plans from the pin and removes the file once
// it installs.
func pinned(m effect.Machine) host.Package {
	return host.Package{M: m, Name: "otelcol-contrib", Source: host.PinnedFile{M: m, Version: version, URL: debURL, SHA256: sums}}
}

// install makes the part that installs the .deb, and the stop of the collector where it installs.
// The package adds the otelcol-contrib user and service. Its postinst enables the service and, on
// a booted machine, starts it with the package's own config, which listens on every address. The
// collector stays off until the role's config is in place, also at boot should a later step fail.
func install(m effect.Machine, u effect.Units, out effect.Out) func(context.Context) ([]host.Resource, error) {
	return func(ctx context.Context) ([]host.Resource, error) {
		ok, err := installed(ctx, m)
		if err != nil || ok {
			return []host.Resource{pinned(m)}, err
		}
		return []host.Resource{pinned(m), stopped{m, u, out}}, nil
	}
}

// stopped stops the collector that the package started and disables it. Without systemd booted it says so
// and plans nothing.
type stopped struct {
	m   effect.Machine
	u   effect.Units
	out effect.Out
}

type stop struct{ u effect.Units }

func (stop) String() string { return "stop and disable otelcol-contrib, which the package started" }

func (c stop) Apply(ctx context.Context) error {
	if err := c.u.Stop(ctx, "otelcol-contrib.service"); err != nil {
		return err
	}
	return c.u.Disable(ctx, "otelcol-contrib.service")
}

func (s stopped) Plan(context.Context) ([]host.Change, error) {
	if !host.Booted(s.m) {
		s.out.Say("collector stop: skipped, systemd is not running")
		return nil, nil
	}
	return []host.Change{stop{s.u}}, nil
}
