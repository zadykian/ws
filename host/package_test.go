package host_test

import (
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

const pkgQ = "dpkg-query -W -f ${Package} ${Version} ${db:Status-Abbrev}\n a"

func packageFixture(dpkg string) (*effecttest.Machine, host.Package) {
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	m.Results[pkgQ] = effecttest.Result{Out: []byte(dpkg)}
	m.Pages["https://x/1.2/a_amd64.deb"] = []byte("deb")
	return m, host.Package{M: m, Name: "a", Source: host.PinnedFile{M: m, Version: "1.2",
		SHA256: map[string]string{"amd64": sumOf("deb")},
		URL:    func(version, arch string) string { return "https://x/" + version + "/a_" + arch + ".deb" }}}
}

func TestPackageInstallsFromItsPinWhereThePinIsMissing(t *testing.T) {
	_, p := packageFixture("")
	changes, err := p.Plan(t.Context())
	if err != nil || len(changes) != 3 || changes[2].String() != "install a 1.2" {
		t.Errorf("plan = %v, %v, want a directory, a fetch and the install", changes, err)
	}
}
