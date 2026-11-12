package host_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

const (
	pkgFile = host.CacheDir + "/a_amd64.deb"
	pkgApt  = "env DEBIAN_FRONTEND=noninteractive apt-get -o DPkg::Lock::Timeout=600 " +
		"-o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold " +
		"install -y --allow-downgrades " + pkgFile
)

const pkgDeb = "dpkg-deb -f " + pkgFile + " Package Version"

const pkgQ = "dpkg-query -W -f ${Package} ${Version} ${db:Status-Abbrev}\n a"

func packageFixture(dpkg string) (*effecttest.Machine, host.Package) {
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	m.Results[pkgQ] = effecttest.Result{Out: []byte(dpkg)}
	m.Results[pkgDeb] = effecttest.Result{Out: []byte("Package: a\nVersion: 1.2\n")}
	m.Results[pkgApt] = effecttest.Result{}
	m.Pages["https://x/1.2/a_amd64.deb"] = []byte("deb")
	return m, host.Package{M: m, Name: "a", Source: host.PinnedFile{M: m, Version: "1.2",
		SHA256: map[string]string{"amd64": sumOf("deb")},
		URL:    func(version, arch string) string { return "https://x/" + version + "/a_" + arch + ".deb" }}}
}

func TestPackageInstallsFromItsPinWhereTheStateDiffers(t *testing.T) {
	for _, dpkg := range []string{"", "a 1.1 ii\n", "a 1.3 ii\n", "a 1.2 rc\n"} {
		m, p := packageFixture(dpkg)
		changes, err := p.Plan(t.Context())
		if err != nil || len(changes) != 3 || changes[2].String() != "install a 1.2" {
			t.Errorf("dpkg %q: plan = %v, %v, want a directory, a fetch and the install", dpkg, changes, err)
		}
		if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasPrefix(c, "download ") }) {
			t.Errorf("dpkg %q: a dry run fetched: %q", dpkg, m.Calls)
		}
	}
}

func TestPackagePlansNothingWhereThePinIsInstalled(t *testing.T) {
	m, p := packageFixture("a 1.2 ii\n")
	if got := applyAll(t, p); got != nil {
		t.Errorf("plan = %q, want none", got)
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, "apt-get") || strings.HasPrefix(c, "download ") }) {
		t.Errorf("calls %q, want no install and no fetch", m.Calls)
	}
}

func TestPackageRemovesAStrayCacheFile(t *testing.T) {
	m, p := packageFixture("a 1.2 ii\n")
	m.Files[pkgFile] = []byte("deb")
	if got := applyAll(t, p); len(got) != 1 || got[0] != "remove "+pkgFile {
		t.Errorf("plan = %q, want the file's removal", got)
	}
}

func TestPackageAppliesAndThenPlansNothing(t *testing.T) {
	m, p := packageFixture("")
	if got := applyAll(t, p); len(got) != 3 {
		t.Fatalf("first plan = %q, want three changes", got)
	}
	if _, ok := m.Files[pkgFile]; ok {
		t.Error("the cache file stayed after the install")
	}
	m.Results[pkgQ] = effecttest.Result{Out: []byte("a 1.2 ii\n")}
	if got := applyAll(t, p); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
}

func TestPackageRefusesAFileOfAnotherNameOrVersion(t *testing.T) {
	for _, field := range []string{"Package: b\nVersion: 1.2\n", "Package: a\nVersion: 1.3\n"} {
		m, p := packageFixture("")
		m.Results[pkgDeb] = effecttest.Result{Out: []byte(field)}
		changes, err := p.Plan(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if err := changes[0].Apply(t.Context()); err != nil {
			t.Fatal(err)
		}
		if err := changes[2].Apply(t.Context()); err == nil {
			t.Errorf("file %q installed, want a refusal", field)
		}
		if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, "apt-get") }) {
			t.Errorf("file %q: apt-get ran: %q", field, m.Calls)
		}
	}
}
