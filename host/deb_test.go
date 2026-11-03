package host_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

func debFixture(installed string) (*effecttest.Machine, host.Deb) {
	m := effecttest.NewMachine()
	m.Results["dpkg-deb -f /tmp/a.deb Package Version"] = effecttest.Result{Out: []byte("Package: a\nVersion: 1.2\n")}
	m.Results["dpkg-query -W -f ${Version} ${db:Status-Abbrev}\n a"] = effecttest.Result{Out: []byte(installed)}
	m.Results["env DEBIAN_FRONTEND=noninteractive apt-get -o DPkg::Lock::Timeout=600 -o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold install -y --allow-downgrades /tmp/a.deb"] = effecttest.Result{}
	return m, host.Deb{M: m, Path: "/tmp/a.deb"}
}

func TestDebInstallsWhereTheVersionDiffers(t *testing.T) {
	for _, installed := range []string{"", "1.1 ii\n", "1.3 ii\n", "1.2 rc\n"} {
		_, d := debFixture(installed)
		if got := applyAll(t, d); len(got) != 1 {
			t.Errorf("installed %q: plan = %q, want one install", installed, got)
		}
	}
}

func TestDebKeepsTheSameVersion(t *testing.T) {
	m, d := debFixture("1.2 ii\n")
	if got := applyAll(t, d); got != nil || slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasPrefix(c, "run env") }) {
		t.Errorf("plan = %q, calls %q; want none", got, m.Calls)
	}
}

func TestDebNeedsAnAbsolutePath(t *testing.T) {
	m, _ := debFixture("")
	if _, err := (host.Deb{M: m, Path: "a.deb"}).Plan(t.Context()); err == nil {
		t.Error("a relative path planned, want an error")
	}
}
