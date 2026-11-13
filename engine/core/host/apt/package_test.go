package apt_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/host/apt"
)

const (
	pkgFile = host.CacheDir + "/a_amd64.deb"
	pkgApt  = "env DEBIAN_FRONTEND=noninteractive apt-get -o DPkg::Lock::Timeout=600 " +
		"-o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold " +
		"install -y --allow-downgrades " + pkgFile
)

const pkgDeb = "dpkg-deb -f " + pkgFile + " Package Version"

// packageFixture is a machine where dpkg lists "a" as the row says, or knows no "a" for a zero row.
func packageFixture(row effecttest.Package) (*effecttest.Machine, apt.Package) {
	m := effecttest.NewMachine()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	m.Packages = effecttest.Packages{}
	if row.Status != "" {
		m.Packages["a"] = row
	}
	m.Results[pkgDeb] = effecttest.Result{Out: []byte("Package: a\nVersion: 1.2\n")}
	m.Results[pkgApt] = effecttest.Result{}
	m.Pages["https://x/1.2/a_amd64.deb"] = []byte("deb")
	return m, apt.Package{M: m, Name: "a", Source: host.PinnedFile{M: m, Version: "1.2",
		SHA256: map[string]string{"amd64": sumOf("deb")},
		URL:    func(version, arch string) string { return "https://x/" + version + "/a_" + arch + ".deb" }}}
}

func TestPackageInstallsFromItsPinWhereTheStateDiffers(t *testing.T) {
	for _, row := range []effecttest.Package{{}, {Version: "1.1", Status: "ii"}, {Version: "1.3", Status: "ii"}, {Version: "1.2", Status: "rc"},
		{Version: "1.2", Status: "hi"}} {
		m, p := packageFixture(row)
		changes, err := p.Plan(t.Context())
		if err != nil || len(changes) != 3 || changes[2].String() != "install a 1.2" {
			t.Errorf("row %+v: plan = %v, %v, want a directory, a fetch and the install", row, changes, err)
		}
		if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasPrefix(c, "download ") }) {
			t.Errorf("row %+v: a dry run fetched: %q", row, m.Calls)
		}
	}
}

func TestPackagePlansNothingWhereThePinIsInstalled(t *testing.T) {
	m, p := packageFixture(effecttest.Package{Version: "1.2", Status: "ii"})
	if got := applyAll(t, p); got != nil {
		t.Errorf("plan = %q, want none", got)
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, "apt-get") || strings.HasPrefix(c, "download ") }) {
		t.Errorf("calls %q, want no install and no fetch", m.Calls)
	}
}

func TestPackageRemovesAStrayCacheFile(t *testing.T) {
	m, p := packageFixture(effecttest.Package{Version: "1.2", Status: "ii"})
	m.Files[pkgFile] = []byte("deb")
	if got := applyAll(t, p); len(got) != 1 || got[0] != "remove "+pkgFile {
		t.Errorf("plan = %q, want the file's removal", got)
	}
}

func TestPackageAppliesAndThenPlansNothing(t *testing.T) {
	m, p := packageFixture(effecttest.Package{})
	if got := applyAll(t, p); len(got) != 3 {
		t.Fatalf("first plan = %q, want three changes", got)
	}
	if _, ok := m.Files[pkgFile]; ok {
		t.Error("the cache file stayed after the install")
	}
	m.Packages["a"] = effecttest.Package{Version: "1.2", Status: "ii"}
	if got := applyAll(t, p); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
}

func TestPackageRefusesAFileOfAnotherNameOrVersion(t *testing.T) {
	for _, field := range []string{"Package: b\nVersion: 1.2\n", "Package: a\nVersion: 1.3\n"} {
		m, p := packageFixture(effecttest.Package{})
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
