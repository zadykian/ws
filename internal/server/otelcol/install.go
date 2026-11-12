package otelcol

import (
	"context"
	"slices"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/host"
)

// installStep is the name of the step that fetches and installs the .deb. The package adds the
// otelcol-contrib user and service. Its postinst enables the service and, on a booted machine,
// starts it with the package's own config, which listens on every address. The collector stays off
// until the role's config is in place, also at boot should a later step fail.
const installStep = "Collector install"

// pinned is the collector's .deb as a package, which plans from the pin and removes the file once
// it installs.
func pinned(m effect.Machine) host.Package {
	return host.Package{M: m, Name: "otelcol-contrib", Source: host.PinnedFile{M: m, Version: version, URL: debURL, SHA256: sums}}
}

// stopped plans the package, and the stop of the collector that it started and enabled where the
// package installs. Without systemd booted it says so and plans no stop.
type stopped struct {
	pkg host.Package
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

func (s stopped) Plan(ctx context.Context) ([]host.Change, error) {
	changes, err := s.pkg.Plan(ctx)
	if err != nil || !slices.ContainsFunc(changes, func(c host.Change) bool { return step.KindOf(c) == step.Install }) {
		return changes, err
	}
	if !host.Booted(s.m) {
		s.out.Say("collector stop: skipped, systemd is not running")
		return changes, nil
	}
	return append(changes, stop{s.u}), nil
}
