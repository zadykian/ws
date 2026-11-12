package fresh

import (
	"path"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

const packageQuery = "dpkg-query -W -f ${Package} ${Version} ${db:Status-Abbrev}\n fresh-editor"

func machine(arch, have string) *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte(arch + "\n")}
	m.Results[packageQuery] = effecttest.Result{Out: []byte(strings.TrimSpace("fresh-editor "+have) + "\n")}
	return m
}

func TestTheURLIsTheReleasesDeb(t *testing.T) {
	want := "https://github.com/sinelaw/fresh/releases/download/v0.5.7/fresh-editor_0.5.7-1_arm64.deb"
	if got := debURL(version, "arm64"); got != want {
		t.Errorf("got %s", got)
	}
}

func TestInstallPlansFromThePinOnANewMachine(t *testing.T) {
	m := machine("amd64", "")
	p, err := (host.Group{Title: installStep, Parts: []host.Resource{pinned(m)}}).Plan(t.Context())
	if err != nil || len(p.Changes) != 3 || p.Changes[2].String() != "install fresh-editor "+version {
		t.Errorf("plan = %v, %v; want the directory, the fetch and the install", p.Changes, err)
	}
}

func TestInstallOnlyClearsTheDebWhereTheVersionIsInstalled(t *testing.T) {
	m := machine("amd64", version+" ii\n")
	m.Files[path.Join(host.CacheDir, debName("amd64"))] = []byte("deb")
	p, err := (host.Group{Title: installStep, Parts: []host.Resource{pinned(m)}}).Plan(t.Context())
	if err != nil || len(p.Changes) != 1 || strings.Contains(p.Changes[0].String(), "install") {
		t.Errorf("plan = %v, %v; want only the .deb's removal", p.Changes, err)
	}
}

func TestInstallFailsOnAnArchitectureWithNoSum(t *testing.T) {
	m := machine("riscv64", "")
	if _, err := (host.Group{Title: installStep, Parts: []host.Resource{pinned(m)}}).Plan(t.Context()); err == nil {
		t.Error("no error")
	}
}
