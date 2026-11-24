package kind

import (
	"slices"
	"strings"
	"testing"
)

// sideSpec holds a unit of the host's alone, one of every other side's, and one of both.
const sideSpec = `
[file.host-only]
path = "/etc/a"
source = "a"
when = { side = "host" }

[file.other]
path = "/etc/b"
source = "a"
when = { side = "!host", set = "yes" }

[file.both]
path = "/etc/c"
source = "a"
`

func TestSideKeepsTheUnitsOfItsSide(t *testing.T) {
	for side, want := range map[string][]string{
		"host":  {"file/both", "file/host-only"},
		"guest": {"file/both", "file/other"},
		"":      {"file/both", "file/other"},
	} {
		e, _, _, _ := testEnv()
		e.Side = side
		got := Names(loadOne(t, e, sideSpec, map[string]string{"a": "x"}))
		slices.Sort(got)
		if !slices.Equal(got, want) {
			t.Errorf("side %q: units = %q, want %q", side, got, want)
		}
	}
}

// sideGatedSpec holds a unit of the host's that another fact turns off, and one that it leaves on.
const sideGatedSpec = `
[file.off]
path = "/etc/a"
source = "a"
when = { side = "host", set = "" }

[file.on]
path = "/etc/b"
source = "a"
when = { side = "host", set = "yes" }
`

func TestSideTurnsAUnitOffAsItPlans(t *testing.T) {
	e, _, _, _ := testEnv()
	e.Side = "host"
	units := loadOne(t, e, sideGatedSpec, map[string]string{"a": "x"})
	byName := map[string]Unit{}
	for _, u := range units {
		byName[u.Name] = u
	}
	if len(byName) != 2 {
		t.Fatalf("units = %q, want both of the host's", Names(units))
	}
	if got := headlines(t, byName["file/off"].Step); got != nil {
		t.Errorf("file/off on the host: plan = %q, want none", got)
	}
	if got := headlines(t, byName["file/on"].Step); got == nil {
		t.Errorf("file/on on the host: plan = none, want the file")
	}
}

func TestSideRefusesAWantOfAnotherForm(t *testing.T) {
	e, _, _, _ := testEnv()
	for _, want := range []string{"Host", "a b", "host/guest"} {
		_, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": gatedFile(`{ side = "` + want + `" }`), "r/a": "x"}))
		if err == nil || !strings.Contains(err.Error(), `when: side "`+want+`" is not of the form`) {
			t.Errorf("load with side %q = %v, want it refused", want, err)
		}
	}
}
