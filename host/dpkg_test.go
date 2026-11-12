package host_test

import (
	"reflect"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/step"
)

func TestPackageReports(t *testing.T) {
	m := effecttest.NewMachine()
	// The table prints what it knows, and ends with exit 1 for the name it doesn't, as dpkg-query does.
	m.Packages = effecttest.Packages{"git": {Version: "1:2.4", Status: "ii"}, "node": {Version: "24.1", Status: "ii"},
		"old": {Version: "1", Status: "rc"}, "never": {Status: "un"}}
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
