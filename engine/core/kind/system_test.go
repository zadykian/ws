package kind

import (
	"reflect"
	"strings"
	"testing"
)

func TestEachKindNamesItsSystems(t *testing.T) {
	for f := range reflect.TypeFor[spec]().Fields() {
		if len(kindsOn[f.Tag.Get("toml")]) == 0 {
			t.Errorf("the kind %s names no system in kindsOn, so no spec loads it", f.Tag.Get("toml"))
		}
	}
}

func TestMacRefusesLinuxKinds(t *testing.T) {
	for _, c := range []struct{ spec, want string }{
		{"[deb.a]\n", "the kind deb does not run on macOS"},
		{"[packages.a]\n", "the kind packages does not run on macOS"},
		{"[dkms.a]\n", "the kind dkms does not run on macOS"},
		{"[service.a]\n", "the kind service does not run on macOS"},
		{"[guard.a]\n", "the kind guard does not run on macOS"},
		{"[binary.a]\n", "the kind binary does not run on macOS"},
		{"[compose.a]\n", "the kind compose does not run on macOS"},
		{"[state.a]\nlinger = \"u\"\n", "linger: on macOS a state takes run alone"},
		{"[state.a]\npurged = \"p\"\n", "purged: on macOS a state takes run alone"},
	} {
		e, _, _, _ := testEnv()
		e.System = MacOS
		if _, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": c.spec})); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q on macOS: load = %v, want %q", c.spec, err, c.want)
		}
	}
}

func TestLinuxRefusesLaunchd(t *testing.T) {
	e, _, _, _ := testEnv()
	_, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": "[launchd.a]\nprefix = \"a.\"\nprogram = \"/a\"\n"}))
	if err == nil || !strings.Contains(err.Error(), "r/units.toml.tmpl: unit launchd/a: the kind launchd does not run on Linux") {
		t.Errorf("load = %v, want launchd refused on Linux", err)
	}
}

func TestMacTakesFileStateRunLaunchd(t *testing.T) {
	e, _, _, _ := testEnv()
	e.System = MacOS
	units, err := Load(e, tree(map[string]string{
		"r/units.toml.tmpl": "[file.a]\npath = \"/etc/a\"\nsource = \"a\"\n\n" +
			"[state.b]\nrun = { user = \"u\", command = [\"/bin/true\"], creates = [\"/b\"] }\n\n" +
			"[launchd.c]\nprefix = \"c.\"\nprogram = \"/c\"\n",
		"r/a": "a",
	}))
	if err != nil || len(units) != 3 {
		t.Errorf("load = %d units, %v; want the three", len(units), err)
	}
}
