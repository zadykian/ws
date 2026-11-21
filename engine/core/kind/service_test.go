package kind

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestServiceEnablesAndRuns(t *testing.T) {
	for state, want := range map[string][]string{
		"running":   {"enable a.service", "start a.service"},
		"stopped":   {"enable a.service", "stop a.service"},
		"triggered": {"enable a.service"},
		"":          {"enable a.service"},
	} {
		e, _, u, _ := testEnv()
		u.States = map[string]effect.UnitState{"a.service": {Active: state == "stopped"}}
		units := loadOne(t, e, "[service.a]\nunit = \"a.service\"\nenabled = true\nrun = \""+state+"\"\n", nil)
		if err := run(t, e, units); err != nil {
			t.Fatalf("run %q: %v", state, err)
		}
		if !slices.Equal(u.Calls, want) {
			t.Errorf("run %q: calls = %q, want %q", state, u.Calls, want)
		}
	}
}

func TestServiceFilesGiveOneRestart(t *testing.T) {
	e, m, u, _ := testEnv()
	m.Dirs["/etc"] = 0o755
	m.Files["/etc/a.conf"], m.Files["/etc/b.conf"] = []byte("old"), []byte("old")
	u.States = map[string]effect.UnitState{"a.service": {Enabled: true, Active: true}}
	units := loadOne(t, e, "[service.a]\nunit = \"a.service\"\nenabled = true\nrun = \"running\"\n"+
		"[[service.a.file]]\npath = \"/etc/a.conf\"\nsource = \"a\"\n"+
		"[[service.a.file]]\npath = \"/etc/b.conf\"\ntemplate = \"b.tmpl\"\n",
		map[string]string{"a": "new", "b.tmpl": "{{.User}}"})
	if err := run(t, e, units); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(u.Calls, []string{"restart a.service"}) {
		t.Errorf("calls = %q, want one restart", u.Calls)
	}
	if string(m.Files["/etc/a.conf"]) != "new" || string(m.Files["/etc/b.conf"]) != "u" {
		t.Errorf("files = %q, %q; want both written", m.Files["/etc/a.conf"], m.Files["/etc/b.conf"])
	}
}

func TestServicePartTakesAnOwner(t *testing.T) {
	e, m, _, _ := testEnv()
	withAccount(m)
	units := loadOne(t, e, "[service.a]\nunit = \"a.service\"\n"+
		"[[service.a.file]]\npath = \"/home/u/.a/a.conf\"\nowner = \"u\"\nmode = 0o600\nsource = \"a\"\n", map[string]string{"a": "x"})
	if err := run(t, e, units); err != nil {
		t.Fatal(err)
	}
	if got := m.Owners["/home/u/.a/a.conf"]; got != [2]int{1001, 1001} || m.Modes["/home/u/.a/a.conf"] != 0o600 {
		t.Errorf("owner = %v, mode %o; want u's at 600", got, m.Modes["/home/u/.a/a.conf"])
	}
}

func TestServiceRestartsOnDrift(t *testing.T) {
	const spec = "[service.a]\nunit = \"a.service\"\nrun = \"running\"\nin_force = [\"show\"]\nloads = [\"load\", \"/etc/a.conf\"]\n" +
		"[[service.a.file]]\npath = \"/etc/a.conf\"\nsource = \"a\"\n"
	for _, c := range []struct {
		inForce string
		want    []string
	}{
		{"same", nil},
		{"other", []string{"restart: what show prints is not what the files load", "note the owed restart of a.service", "restart a.service"}},
	} {
		e, m, u, _ := testEnv()
		m.Dirs["/etc"] = 0o755
		m.Files["/etc/a.conf"] = []byte("x")
		u.States = map[string]effect.UnitState{"a.service": {Active: true}}
		m.Results["load /etc/a.conf"] = effecttest.Result{Out: []byte("same")}
		m.Results["show"] = effecttest.Result{Out: []byte(c.inForce)}
		units := loadOne(t, e, spec, map[string]string{"a": "x"})
		got := headlines(t, units[0].Step)
		slices.Sort(got)
		slices.Sort(c.want)
		if !slices.Equal(got, c.want) {
			t.Errorf("in force %q: plan = %q, want %q", c.inForce, got, c.want)
		}
		// The drift stays until the restart, which must not wait for it.
		p, err := units[0].Step.Plan(t.Context())
		if err == nil {
			err = units[0].Step.Apply(t.Context(), p)
		}
		if restarted := slices.Contains(u.Calls, "restart a.service"); err != nil || restarted != (c.want != nil) {
			t.Errorf("in force %q: apply = %v, calls %q; want a restart %t", c.inForce, err, u.Calls, c.want != nil)
		}
	}
}

func TestServiceRefusesFieldsNoPlanTakes(t *testing.T) {
	e, _, _, _ := testEnv()
	for spec, want := range map[string]string{
		"unit = \"a\"\n":                                            "type's suffix",
		"unit = \"a.service\"\nrun = \"up\"\n":                      `run "up" is none of`,
		"unit = \"a.service\"\nin_force = [\"show\"]\n":             "in_force goes with loads",
		"unit = \"a.service\"\n[[service.a.file]]\npath = \"/a\"\n": "file 1: a file takes a template or a source",
		"unit = \"a.service\"\n[[service.a.file]]\npath = \"/a\"\nsource = \"f\"\nservice = \"b.service\"\n": "service.a.file.service",
		"unit = \"a.service\"\n[[service.a.file]]\npath = \"/a\"\nsource = \"f\"\nafter = \"file/b\"\n":      "service.a.file.after",
	} {
		_, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": "[service.a]\n" + spec, "r/f": ""}))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("load of\n%s= %v, want %q", spec, err, want)
		}
	}
}

func TestServiceFilesInOneNewDirectoryMakeItOnce(t *testing.T) {
	e, m, u, _ := testEnv()
	m.Dirs["/etc"] = 0o755
	u.States = map[string]effect.UnitState{"a.service": {Enabled: true, Active: true}}
	units := loadOne(t, e, "[service.a]\nunit = \"a.service\"\nenabled = true\nrun = \"running\"\n"+
		"[[service.a.file]]\npath = \"/etc/a.d/a.conf\"\nsource = \"a\"\n"+
		"[[service.a.file]]\npath = \"/etc/a.d/b.conf\"\nsource = \"b\"\n", map[string]string{"a": "a", "b": "b"})
	if err := run(t, e, units); err != nil {
		t.Fatal(err)
	}
	if string(m.Files["/etc/a.d/a.conf"]) != "a" || string(m.Files["/etc/a.d/b.conf"]) != "b" || m.Dirs["/etc/a.d"] != 0o755 {
		t.Errorf("files = %q, %q, dir %o; want both written in the new directory", m.Files["/etc/a.d/a.conf"], m.Files["/etc/a.d/b.conf"], m.Dirs["/etc/a.d"])
	}
}

// A file that goes waits for the files written after it in the spec: a refused check of one of
// them keeps it.
func TestServiceRemovesAFileAfterThoseItWrites(t *testing.T) {
	e, m, u, _ := testEnv()
	m.Dirs["/etc"] = 0o755
	m.Files["/etc/a.env"], m.Files["/etc/b.conf"] = []byte("A=1\n"), []byte("old")
	m.Results["refuse /etc/b.conf.ws-check"] = effecttest.Result{Err: errors.New("exit 1")}
	u.States = map[string]effect.UnitState{"a.service": {Enabled: true, Active: true}}
	units := loadOne(t, e, "[service.a]\nunit = \"a.service\"\nenabled = true\nrun = \"running\"\n"+
		"[[service.a.file]]\npath = \"/etc/a.env\"\nabsent = true\n"+
		"[[service.a.file]]\npath = \"/etc/b.conf\"\nsource = \"b\"\ncheck = [\"refuse\", \"{}\"]\n",
		map[string]string{"b": "new"})
	if err := run(t, e, units); err == nil {
		t.Fatal("a file its check refuses was written")
	}
	if _, ok := m.Files["/etc/a.env"]; !ok || string(m.Files["/etc/b.conf"]) != "old" {
		t.Errorf("a.env there = %v, b.conf = %q; want both as they were", ok, m.Files["/etc/b.conf"])
	}
}
