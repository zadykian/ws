package apt_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource/apt"
)

func TestPurgedRemovesWhatIsThereAndWhatLeftItsSettings(t *testing.T) {
	m := effecttest.NewMachine()
	m.Packages = effecttest.Packages{"lxd-installer": {Version: "1", Status: "rc"}, "snapd": {Version: "2", Status: "ii"}}
	m.Results["env DEBIAN_FRONTEND=noninteractive apt-get -o DPkg::Lock::Timeout=600 -o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold purge -y lxd-installer snapd"] = effecttest.Result{}
	got := applyAll(t, apt.Purged{M: m, Names: []string{"lxd-installer", "snapd"}})
	if len(got) != 1 || got[0] != "purge lxd-installer snapd" {
		t.Errorf("plan = %q, want one purge of both", got)
	}
	if !slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, "purge -y lxd-installer snapd") }) {
		t.Errorf("calls = %q, want apt-get purge", m.Calls)
	}
}

func TestPurgedLeavesAPackageThatIsNotThere(t *testing.T) {
	m := effecttest.NewMachine()
	m.Packages = effecttest.Packages{"snapd": {Status: "un"}}
	if got := applyAll(t, apt.Purged{M: m, Names: []string{"lxd-installer", "snapd"}}); got != nil {
		t.Errorf("plan = %q, want none", got)
	}
}

func TestPurgedWithNoNamesPlansNothingAndRunsNothing(t *testing.T) {
	m := effecttest.NewMachine()
	m.Packages = effecttest.Packages{"snapd": {Version: "2", Status: "ii"}}
	if got := applyAll(t, apt.Purged{M: m}); got != nil || len(m.Calls) != 0 {
		t.Errorf("plan = %q, calls %q, want none", got, m.Calls)
	}
}
