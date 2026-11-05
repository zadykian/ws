package host_test

import (
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

const purgeQuery = "dpkg-query -W -f ${Package} ${db:Status-Abbrev}\n lxd-installer snapd"

func TestPurgedRemovesWhatIsThereAndWhatLeftItsSettings(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[purgeQuery] = effecttest.Result{Out: []byte("lxd-installer rc\nsnapd ii\n")}
	m.Results["env DEBIAN_FRONTEND=noninteractive apt-get -o DPkg::Lock::Timeout=600 -o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold purge -y lxd-installer snapd"] = effecttest.Result{}
	got := applyAll(t, host.Purged{M: m, Names: []string{"lxd-installer", "snapd"}})
	if len(got) != 1 || got[0] != "purge lxd-installer snapd" {
		t.Errorf("plan = %q, want one purge of both", got)
	}
	if !slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, "purge -y lxd-installer snapd") }) {
		t.Errorf("calls = %q, want apt-get purge", m.Calls)
	}
}

func TestPurgedLeavesAPackageThatIsNotThere(t *testing.T) {
	exit1 := exec.Command("sh", "-c", "exit 1").Run()
	m := effecttest.NewMachine()
	m.Results[purgeQuery] = effecttest.Result{Out: []byte("snapd n\n"), Err: exit1}
	if got := applyAll(t, host.Purged{M: m, Names: []string{"lxd-installer", "snapd"}}); got != nil {
		t.Errorf("plan = %q, want none", got)
	}
}
