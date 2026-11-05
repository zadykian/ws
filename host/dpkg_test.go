package host_test

import (
	"os/exec"
	"reflect"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/step"
)

func TestPackageReports(t *testing.T) {
	const query = "dpkg-query -W -f ${Package} ${Version} ${db:Status-Abbrev}\n "
	m := effecttest.NewMachine()
	// dpkg-query prints what it knows, and exits with 1 for the name it doesn't.
	known := effecttest.Result{Out: []byte("git 1:2.4 ii\nnode 24.1 ii\nold 1 rc\nnever  un\n"),
		Err: exec.Command("sh", "-c", "exit 1").Run()}
	m.Results[query+"git old gone"] = known
	item := func(name, want, found, conflict string) step.Item {
		return step.Item{Kind: step.PackageItem, Name: name, Want: want, Found: found, Conflict: conflict}
	}
	got, err := host.Purged{M: m, Names: []string{"git", "old", "gone"}}.Report(t.Context())
	want := []step.Item{item("git", "missing", "1:2.4", "installed, and ws purges it"),
		item("old", "missing", "settings left", ""), item("gone", "missing", "missing", "")}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("items = %+v, %v, want %+v", got, err, want)
	}
}
