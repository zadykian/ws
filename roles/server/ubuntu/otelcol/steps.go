package otelcol

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/host/account"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// Step names. Those after the install wait on the steps before them.
const (
	exporterStep = "node_exporter"
	configStep   = "Collector configuration"
	healthStep   = "Collector health"
)

// Steps returns the role's steps in order: node_exporter, the .deb's install, the
// collector's configuration and units, and the check of its answer. node_exporter comes first, so
// that the collector's first scrape finds it. The role takes no secrets.
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, u effect.Units, out effect.Out) []step.Step {
	return steps(v, m, u, out, healthURL)
}

// steps is Steps with the health URL given, for a test to fake.
func steps(v values.Values, m effect.Machine, u effect.Units, out effect.Out, health string) []step.Step {
	return []step.Step{
		host.Lazy{Title: exporterStep, Parts: exporter(m, u, out)},
		host.Group{Title: installStep, Parts: []host.Resource{stopped{pinned(m), m, u, out}}},
		host.Lazy{Title: configStep, After: installStep, Parts: configuration(m, u, out, account.User{Name: v.User, Home: v.Home})},
		host.Group{Title: healthStep, After: configStep, Parts: []host.Resource{
			answer{m: m, out: out, url: health, tries: healthTries, wait: healthWait},
		}},
	}
}
