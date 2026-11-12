package otelcol

import (
	"crypto/sha256"
	"fmt"
	"path"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

func installGroup(m *effecttest.Machine, u effect.Units, out effect.Out) host.Group {
	return host.Group{Title: installStep, Parts: []host.Resource{stopped{pinned(m), m, u, out}}}
}

func TestInstallRemovesAStrayDeb(t *testing.T) {
	m := machine(effecttest.Package{Version: version, Status: "ii"})
	m.Files[path.Join(host.CacheDir, debName("amd64"))] = []byte("deb")
	p, err := installGroup(m, &effecttest.Units{}, &effecttest.Out{}).Plan(t.Context())
	if err != nil || len(p.Changes) != 1 || !strings.HasPrefix(p.Changes[0].String(), "remove ") {
		t.Errorf("plan with a stray .deb = %v, %v; want its removal", p.Changes, err)
	}
}

func TestOtherVersionDownloadsChecksInstallsAndRemoves(t *testing.T) {
	m := machine(effecttest.Package{Version: "0.100.0", Status: "ii"})
	m.Pages[debURL(version, "amd64")] = []byte("deb")
	old := sums["amd64"]
	sums["amd64"] = fmt.Sprintf("%x", sha256.Sum256([]byte("deb")))
	t.Cleanup(func() { sums["amd64"] = old })
	m.Results["dpkg-deb -f "+path.Join(host.CacheDir, debName("amd64"))+" Package Version"] = effecttest.Result{Out: []byte("Package: otelcol-contrib\nVersion: " + version + "\n")}
	m.Results["env DEBIAN_FRONTEND=noninteractive apt-get -o DPkg::Lock::Timeout=600 "+
		"-o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold install -y --allow-downgrades "+path.Join(host.CacheDir, debName("amd64"))] = effecttest.Result{}
	m.Dirs["/run/systemd/system"] = 0o755
	u := &effecttest.Units{}
	// The table keeps its rows through apt-get, so the install is planned and applied once, not re-planned.
	in := installGroup(m, u, &effecttest.Out{})
	p, err := in.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := in.Apply(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Files[path.Join(host.CacheDir, debName("amd64"))]; ok {
		t.Error("the .deb stays in /var/cache")
	}
	if got := strings.Join(u.Calls, ","); got != "stop otelcol-contrib.service,disable otelcol-contrib.service" {
		t.Errorf("unit calls = %q, want a stop and a disable", got)
	}
}

func TestInstallPlansFromThePinOnANewMachine(t *testing.T) {
	m := machine(effecttest.Package{})
	m.Dirs["/run/systemd/system"] = 0o755
	p, err := installGroup(m, &effecttest.Units{}, &effecttest.Out{}).Plan(t.Context())
	if err != nil || len(p.Changes) != 4 || p.Changes[2].String() != "install otelcol-contrib "+version {
		t.Errorf("plan = %v, %v; want the directory, the fetch, the install and the stop", p.Changes, err)
	}
}
