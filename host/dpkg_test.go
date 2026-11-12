package host_test

import (
	"reflect"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/host"
)

func TestPackageReports(t *testing.T) {
	m := effecttest.NewMachine()
	// The table prints what it knows, and ends with exit 1 for the name it doesn't, as dpkg-query does.
	m.Packages = effecttest.Packages{"git": {Version: "1:2.4", Status: "ii"}, "node": {Version: "24.1", Status: "ii"},
		"old": {Version: "1", Status: "rc"}, "never": {Status: "un"},
		"half": {Version: "1.2", Status: "iU"}, "held": {Version: "3", Status: "hi"}}
	item := func(name, want, found, conflict string) step.Item {
		return step.Item{Kind: step.PackageItem, Name: name, Want: want, Found: found, Conflict: conflict}
	}
	tests := []struct {
		r    step.Reporter
		want []step.Item
	}{
		{host.Packages{M: m, Names: []string{"git", "node=24.*", "old", "never", "gone"}}, []step.Item{
			item("git", "installed", "1:2.4", ""), item("node", "24.*", "24.1", ""),
			item("old", "installed", "settings left", ""), item("never", "installed", "missing", ""),
			item("gone", "installed", "missing", "")}},
		{host.Packages{M: m, Names: []string{"half", "held"}}, []step.Item{
			item("half", "installed", "1.2 (iU)", ""), item("held", "installed", "3 (hi)", "")}},
		{host.Purged{M: m, Names: []string{"git", "old", "gone"}}, []step.Item{
			item("git", "missing", "1:2.4", "installed, and ws purges it"),
			item("old", "missing", "settings left", ""), item("gone", "missing", "missing", "")}},
	}
	for _, tt := range tests {
		got, err := tt.r.Report(t.Context())
		if err != nil || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("items = %+v, %v, want %+v", got, err, tt.want)
		}
	}
}

func TestPackageReportsKeepTheInstalledLineOfTwoArchitectures(t *testing.T) {
	for _, out := range []string{"libc6 2.40 ii \nlibc6 2.40 un \n", "libc6 2.40 un \nlibc6 2.40 ii \n"} {
		m := effecttest.NewMachine()
		m.Results["dpkg-query -W -f ${Package} ${Version} ${db:Status-Abbrev}\n libc6"] = effecttest.Result{Out: []byte(out)}
		got, err := host.Packages{M: m, Names: []string{"libc6"}}.Report(t.Context())
		if err != nil || len(got) != 1 || got[0].Found != "2.40" {
			t.Errorf("out %q: items = %+v, %v, want the version", out, got, err)
		}
	}
}

func TestPackagesReportNoNamesRunsNoDpkg(t *testing.T) {
	m := effecttest.NewMachine()
	got, err := host.Packages{M: m}.Report(t.Context())
	if err != nil || len(got) != 0 || len(m.Calls) != 0 {
		t.Errorf("items = %+v, %v, calls %q, want none and no call", got, err, m.Calls)
	}
}

func TestPackageStatesReadsPatternsInOneRunAndNoneRunsNothing(t *testing.T) {
	m := effecttest.NewMachine()
	m.Packages = effecttest.Packages{"linux-image-generic": {Version: "6.8", Status: "ii"},
		"linux-image-generic-hwe-24.04": {Version: "6.11", Status: "rc"}}
	got, err := host.PackageStates(t.Context(), m, "linux-image-generic*", "absent")
	want := map[string]host.PackageState{"linux-image-generic": {Version: "6.8", Status: "ii"},
		"linux-image-generic-hwe-24.04": {Version: "6.11", Status: "rc"}}
	if err != nil || !reflect.DeepEqual(got, want) || len(m.Calls) != 1 {
		t.Errorf("states = %+v, %v, calls %q, want %+v from one run", got, err, m.Calls, want)
	}
	if !got["linux-image-generic"].Installed() || got["linux-image-generic-hwe-24.04"].Installed() {
		t.Errorf("states = %+v, want ii alone installed", got)
	}
	m.Calls = nil
	if got, err := host.PackageStates(t.Context(), m); err != nil || len(got) != 0 || len(m.Calls) != 0 {
		t.Errorf("no pattern: states = %+v, %v, calls %q, want none", got, err, m.Calls)
	}
}
