package fresh

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

const query = "dpkg-query -W -f ${Version} ${db:Status-Abbrev}\n fresh-editor"

func machine(arch, have string) *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte(arch + "\n")}
	m.Results[query] = effecttest.Result{Out: []byte(have)}
	return m
}

func TestDownloadPlansNothingWhereTheVersionIsInstalled(t *testing.T) {
	lazy := host.Lazy{Title: downloadStep, Parts: download(machine("amd64", version+" ii\n"))}
	if p, err := lazy.Plan(t.Context()); err != nil || len(p.Changes) != 0 {
		t.Errorf("plan = %v, %v; want none", p.Changes, err)
	}
}

func TestDownloadFetchesWhereAnotherVersionIsInstalled(t *testing.T) {
	for _, arch := range []string{"amd64", "arm64"} {
		lazy := host.Lazy{Title: downloadStep, Parts: download(machine(arch, "0.5.6-1 ii\n"))}
		p, err := lazy.Plan(t.Context())
		if err != nil || len(p.Changes) != 1 || !strings.Contains(p.Changes[0].String(), debURL(arch)) {
			t.Errorf("%s: plan = %v, %v; want the release's .deb", arch, p.Changes, err)
		}
	}
}

func TestDownloadFailsOnAnArchitectureWithNoSum(t *testing.T) {
	lazy := host.Lazy{Title: downloadStep, Parts: download(machine("riscv64", ""))}
	if _, err := lazy.Plan(t.Context()); err == nil {
		t.Error("no error")
	}
}

func TestTheURLIsTheReleasesDeb(t *testing.T) {
	want := "https://github.com/sinelaw/fresh/releases/download/v0.5.7/fresh-editor_0.5.7-1_arm64.deb"
	if got := debURL("arm64"); got != want {
		t.Errorf("got %s", got)
	}
}

func TestInstallOnlyClearsTheDebWhereTheVersionIsInstalled(t *testing.T) {
	m := machine("amd64", version+" ii\n")
	m.Files[debPath("amd64")] = []byte("deb")
	lazy := host.Lazy{Title: installStep, Parts: install(m)}
	p, err := lazy.Plan(t.Context())
	if err != nil || len(p.Changes) != 1 || strings.Contains(p.Changes[0].String(), "install") {
		t.Errorf("plan = %v, %v; want only the .deb's removal", p.Changes, err)
	}
}
