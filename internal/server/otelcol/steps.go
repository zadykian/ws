package otelcol

import (
	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/server/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

// Step names. Those after the download wait on the steps before them.
const (
	exporterStep = "node_exporter"
	configStep   = "Collector configuration"
	healthStep   = "Collector health"
)

// Steps returns the role's steps in order: node_exporter, the .deb's download and install, the
// collector's configuration and units, and the check of its answer. node_exporter comes first, so
// that the collector's first scrape finds it. The role takes no secrets.
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, out effect.Out) []step.Step {
	return steps(v, m, effect.OSUnits{}, out, healthURL)
}

// steps is Steps with the units and the health URL given, for a test to fake.
func steps(v values.Values, m effect.Machine, u effect.Units, out effect.Out, health string) []step.Step {
	return []step.Step{
		host.Lazy{Title: exporterStep, Parts: exporter(m, u, out)},
		host.Lazy{Title: downloadStep, Parts: download(m)},
		host.Lazy{Title: installStep, After: downloadStep, Parts: install(m, u, out)},
		host.Lazy{Title: configStep, After: installStep, Parts: configuration(m, u, out, v.Home)},
		host.Group{Title: healthStep, After: configStep, Parts: []host.Resource{
			answer{m: m, out: out, url: health, tries: healthTries, wait: healthWait},
		}},
	}
}
