package otelcol

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/step"
)

func TestInstallRemovesAStrayDeb(t *testing.T) {
	m := machine(version + " ii\n")
	m.Files[debPath("amd64")] = []byte("deb")
	p, err := host.Lazy{Title: installStep, Parts: install(m, &effecttest.Units{})}.Plan(t.Context())
	if err != nil || len(p.Changes) != 1 || !strings.HasPrefix(p.Changes[0].String(), "remove ") {
		t.Errorf("plan with a stray .deb = %v, %v; want its removal", p.Changes, err)
	}
}

func TestOtherVersionDownloadsChecksInstallsAndRemoves(t *testing.T) {
	m := machine("0.100.0 ii\n")
	m.Pages[debURL("amd64")] = []byte("deb")
	old := sums["amd64"]
	sums["amd64"] = fmt.Sprintf("%x", sha256.Sum256([]byte("deb")))
	t.Cleanup(func() { sums["amd64"] = old })
	m.Results["dpkg-deb -f "+debPath("amd64")+" Package Version"] = effecttest.Result{Out: []byte("Package: otelcol-contrib\nVersion: " + version + "\n")}
	m.Results["env DEBIAN_FRONTEND=noninteractive apt-get -o DPkg::Lock::Timeout=600 "+
		"-o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold install -y --allow-downgrades "+debPath("amd64")] = effecttest.Result{}
	m.Dirs["/run/systemd/system"] = 0o755
	u := &effecttest.Units{}
	dl := host.Lazy{Title: downloadStep, Parts: download(m)}
	if changed, err := step.Run(t.Context(), []step.Step{dl}, &effecttest.Out{}, false); err != nil || !changed {
		t.Fatalf("download = %v, %v; want the file", changed, err)
	}
	// The fake keeps no dpkg database, so the install is planned and applied once, not re-planned.
	in := host.Lazy{Title: installStep, Parts: install(m, u)}
	p, err := in.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := in.Apply(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Files[debPath("amd64")]; ok {
		t.Error("the .deb stays in /var/cache")
	}
	if got := strings.Join(u.Calls, ","); got != "stop otelcol-contrib.service,disable otelcol-contrib.service" {
		t.Errorf("unit calls = %q, want a stop and a disable", got)
	}
}
