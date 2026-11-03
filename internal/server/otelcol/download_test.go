package otelcol

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

const query = "dpkg-query -W -f ${Version} ${db:Status-Abbrev}\n otelcol-contrib"

func machine(have string) *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	m.Results[query] = effecttest.Result{Out: []byte(have)}
	return m
}

func TestDownloadPlansNothingWhereTheVersionIsInstalled(t *testing.T) {
	lazy := host.Lazy{Title: downloadStep, Parts: download(machine(version + " ii\n"))}
	if p, err := lazy.Plan(t.Context()); err != nil || len(p.Changes) != 0 {
		t.Errorf("plan = %v, %v; want none", p.Changes, err)
	}
}

func TestDownloadFetchesWhereAnotherVersionIsInstalled(t *testing.T) {
	lazy := host.Lazy{Title: downloadStep, Parts: download(machine("0.100.0 ii\n"))}
	p, err := lazy.Plan(t.Context())
	if err != nil || len(p.Changes) != 1 || !strings.Contains(p.Changes[0].String(), debURL("amd64")) {
		t.Errorf("plan = %v, %v; want the release's .deb", p.Changes, err)
	}
}
