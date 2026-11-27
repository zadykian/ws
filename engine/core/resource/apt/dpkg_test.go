package apt_test

import (
	"reflect"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource/apt"
)

func TestPackageStatesKeepTheInstalledLineOfTwoArchitectures(t *testing.T) {
	for _, out := range []string{"libc6 2.40 ii \nlibc6 2.40 un \n", "libc6 2.40 un \nlibc6 2.40 ii \n"} {
		m := effecttest.NewMachine()
		m.Results["dpkg-query -W -f ${Package} ${Version} ${db:Status-Abbrev}\n libc6"] = effecttest.Result{Out: []byte(out)}
		got, err := apt.PackageStates(t.Context(), m, "libc6")
		if err != nil || got["libc6"].Version != "2.40" || !got["libc6"].Installed() {
			t.Errorf("out %q: states = %+v, %v, want the installed version", out, got, err)
		}
	}
}

func TestPackageStatesReadsPatternsInOneRunAndNoneRunsNothing(t *testing.T) {
	m := effecttest.NewMachine()
	m.Packages = effecttest.Packages{"linux-image-generic": {Version: "6.8", Status: "ii"},
		"linux-image-generic-hwe-24.04": {Version: "6.11", Status: "rc"}}
	got, err := apt.PackageStates(t.Context(), m, "linux-image-generic*", "absent")
	want := map[string]apt.PackageState{"linux-image-generic": {Version: "6.8", Status: "ii"},
		"linux-image-generic-hwe-24.04": {Version: "6.11", Status: "rc"}}
	if err != nil || !reflect.DeepEqual(got, want) || len(m.Calls) != 1 {
		t.Errorf("states = %+v, %v, calls %q, want %+v from one run", got, err, m.Calls, want)
	}
	if !got["linux-image-generic"].Installed() || got["linux-image-generic-hwe-24.04"].Installed() {
		t.Errorf("states = %+v, want ii alone installed", got)
	}
	m.Calls = nil
	if got, err := apt.PackageStates(t.Context(), m); err != nil || len(got) != 0 || len(m.Calls) != 0 {
		t.Errorf("no pattern: states = %+v, %v, calls %q, want none", got, err, m.Calls)
	}
}
