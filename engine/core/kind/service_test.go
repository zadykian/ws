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
		e, _, u, _ := testEnv(true)
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

func TestServiceSkipsWithoutSystemd(t *testing.T) {
	e, m, u, out := testEnv(false)
	m.Dirs["/etc"] = 0o755
	units := loadOne(t, e, "[service.a]\nunit = \"a.service\"\nenabled = true\nrun = \"running\"\n"+
		"[[service.a.file]]\npath = \"/etc/a.conf\"\nsource = \"a\"\n", map[string]string{"a": "x"})
	if got := headlines(t, units[0].Step); !slices.Equal(got, []string{"create /etc/a.conf"}) {
		t.Errorf("plan = %q, want the file alone", got)
	}
	if u.Calls != nil || !slices.Equal(out.Lines, []string{"say: a.service: skipped, systemd is not running"}) {
		t.Errorf("calls = %q, lines %q; want none, and the skip said once", u.Calls, out.Lines)
	}
}

func TestServiceFilesGiveOneRestart(t *testing.T) {
	e, m, u, _ := testEnv(true)
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
	e, m, _, _ := testEnv(false)
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
		e, m, u, _ := testEnv(true)
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
	e, _, _, _ := testEnv(false)
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

// userSystemctl is how a unit of u's own systemd is called, as u in u's session.
const userSystemctl = "runuser -u u -- env -C /home/u HOME=/home/u env XDG_RUNTIME_DIR=/run/user/1001 " +
	"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1001/bus systemctl --user "

// userService is a service of u's own systemd, x.service, with two files of u's.
const userService = "[service.a]\nunit = \"x.service\"\nuser = \"u\"\nrun = \"running\"\n" +
	"[[service.a.file]]\npath = \"/home/u/.config/a.conf\"\nowner = \"u\"\nsource = \"a\"\n" +
	"[[service.a.file]]\npath = \"/home/u/.config/b.conf\"\nowner = \"u\"\nsource = \"a\"\n"

// userServiceEnv is a booted machine with u, whose manager is up and runs x.service as state
// says, unless state is "": then x.service has no unit file.
func userServiceEnv(state string) (Env, *effecttest.Machine) {
	e, m, _, _ := testEnv(true)
	withAccount(m)
	m.Dirs["/home/u/.config"] = 0o755
	m.Owners["/home/u/.config"] = [2]int{1001, 1001}
	for _, c := range []string{"daemon-reload", "start x.service", "restart x.service"} {
		m.Results[userSystemctl+c] = effecttest.Result{}
	}
	if state != "" {
		m.Files["/home/u/.config/systemd/user/x.service"] = nil
		m.Dirs["/run/user/1001/systemd/private"] = 0o755
		m.Results[userSystemctl+"show --property=ActiveState --value x.service"] = effecttest.Result{Out: []byte(state + "\n")}
	}
	return e, m
}

// userCalls returns the verbs that went to u's systemctl.
func userCalls(m *effecttest.Machine) []string {
	var got []string
	for _, c := range m.Calls {
		if verb, ok := strings.CutPrefix(c, "root "+userSystemctl); ok && !strings.HasPrefix(verb, "show ") {
			got = append(got, verb)
		}
	}
	return got
}

func TestServiceOfAUserRestartsOnce(t *testing.T) {
	e, m := userServiceEnv("active")
	m.Files["/home/u/.config/a.conf"], m.Files["/home/u/.config/b.conf"] = []byte("old"), []byte("old")
	units := loadOne(t, e, userService, map[string]string{"a": "new"})
	if err := run(t, e, units); err != nil {
		t.Fatal(err)
	}
	if got, want := userCalls(m), []string{"daemon-reload", "restart x.service"}; !slices.Equal(got, want) {
		t.Errorf("u's systemctl = %q, want %q", got, want)
	}
	if string(m.Files["/home/u/.config/a.conf"]) != "new" || string(m.Files["/home/u/.config/b.conf"]) != "new" {
		t.Errorf("files = %q, %q; want both written", m.Files["/home/u/.config/a.conf"], m.Files["/home/u/.config/b.conf"])
	}
}

func TestServiceOfAUserStartsWhenDown(t *testing.T) {
	e, m := userServiceEnv("inactive")
	for _, f := range []string{"/home/u/.config/a.conf", "/home/u/.config/b.conf"} {
		m.Files[f], m.Owners[f] = []byte("new"), [2]int{1001, 1001}
	}
	units := loadOne(t, e, userService, map[string]string{"a": "new"})
	if got := headlines(t, units[0].Step); !slices.Equal(got, []string{"start u's x.service"}) {
		t.Errorf("plan = %q, want a start", got)
	}
}

func TestServiceOfAUserWaitsForItsUnitFile(t *testing.T) {
	e, m := userServiceEnv("")
	units := loadOne(t, e, userService, map[string]string{"a": "new"})
	if err := run(t, e, units); err != nil {
		t.Fatal(err)
	}
	if got := userCalls(m); got != nil || string(m.Files["/home/u/.config/b.conf"]) != "new" {
		t.Errorf("u's systemctl = %q, b.conf %q; want the files written and no call", got, m.Files["/home/u/.config/b.conf"])
	}
}

// A part that is the unit's own file goes before the start, in the run that writes it, which then
// plans nothing more.
func TestServiceOfAUserStartsTheUnitFileItWrites(t *testing.T) {
	const unitFile = "/home/u/.config/systemd/user/x.service"
	e, m := userServiceEnv("")
	m.Dirs["/run/user/1001/systemd/private"] = 0o755
	m.Results[userSystemctl+"show --property=ActiveState --value x.service"] = effecttest.Result{Out: []byte("active\n")}
	spec := userService + "[[service.a.file]]\npath = \"" + unitFile + "\"\nowner = \"u\"\nsource = \"a\"\n"
	units := loadOne(t, e, spec, map[string]string{"a": "new"})
	if got := headlines(t, units[0].Step); len(got) == 0 || got[len(got)-1] != "start u's x.service" ||
		!slices.Contains(got, "create "+unitFile) {
		t.Fatalf("plan = %q, want the unit's file, then its start", got)
	}
	if err := run(t, e, units); err != nil {
		t.Fatal(err)
	}
	if got, want := userCalls(m), []string{"daemon-reload", "start x.service"}; !slices.Equal(got, want) ||
		string(m.Files[unitFile]) != "new" {
		t.Errorf("u's systemctl = %q, the unit's file %q; want %q after the file", got, m.Files[unitFile], want)
	}
}

func TestServiceOfAUserSkipsUnbooted(t *testing.T) {
	e, m, _, out := testEnv(false)
	withAccount(m)
	m.Dirs["/home/u/.config"] = 0o755
	m.Owners["/home/u/.config"] = [2]int{1001, 1001}
	units := loadOne(t, e, userService, map[string]string{"a": "new"})
	want := []string{"create /home/u/.config/a.conf", "create /home/u/.config/b.conf"}
	if got := headlines(t, units[0].Step); !slices.Equal(got, want) ||
		!slices.Equal(out.Lines, []string{"say: u's x.service: skipped, systemd is not running"}) {
		t.Errorf("plan = %q, lines %q; want %q, and the skip said once", got, out.Lines, want)
	}
}

func TestServiceOfAUserRefusesFieldsNoPlanTakes(t *testing.T) {
	e, _, _, _ := testEnv(false)
	const head = "[service.a]\nunit = \"x.service\"\n"
	const part = "[[service.a.file]]\npath = \"/home/u/a\"\nsource = \"a\"\n"
	for _, c := range []struct{ fields, want string }{
		{"user = \"u\"\nrun = \"running\"\nenabled = true\n", "takes no enabled"},
		{"user = \"u\"\nrun = \"running\"\nreload = true\n", "takes no enabled"},
		{"user = \"u\"\nrun = \"running\"\nin_force = [\"a\"]\nloads = [\"b\"]\n", "takes no enabled"},
		{"user = \"u\"\nrun = \"stopped\"\n", `takes run = "running"`},
		{"user = \"u\"\n", `takes run = "running"`},
		{"user = \"root\"\nrun = \"running\"\n", "other than root"},
		{"user = \"a b\"\nrun = \"running\"\n", `user "a b" is no user's name`},
		{"user = \"u\"\nrun = \"running\"\n" + part + "owner = \"v\"\n", `owner "v" is not the unit's user "u"`},
		{"user = \"u\"\nrun = \"running\"\n" + part, `owner "" is not the unit's user "u"`},
	} {
		_, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": head + c.fields, "r/a": "x"}))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("load of\n%s= %v, want %q", c.fields, err, c.want)
		}
	}
}

func TestServiceFilesInOneNewDirectoryMakeItOnce(t *testing.T) {
	e, m, u, _ := testEnv(true)
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
	e, m, u, _ := testEnv(true)
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
