package kind

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

// withAccount gives the machine the account u, uid 1001, at home /home/u.
func withAccount(m *effecttest.Machine) {
	m.Results["id -u u"] = effecttest.Result{Out: []byte("1001\n")}
	m.Results["id -g u"] = effecttest.Result{Out: []byte("1001\n")}
	m.Results["getent passwd u"] = effecttest.Result{Out: []byte("u:x:1001:1001::/home/u:/bin/bash\n")}
	m.Dirs["/home/u"] = 0o750
	m.Owners["/home/u"] = [2]int{1001, 1001}
}

// run runs the units as a write does, and returns its error.
func run(t *testing.T, e Env, units []Unit) error {
	_, err := step.Run(t.Context(), Steps(units), e.Out, false)
	return err
}

func TestFileWritesItsTemplate(t *testing.T) {
	e, m, _, _ := testEnv(false)
	m.Dirs["/etc"] = 0o755
	units := loadOne(t, e, "[file.a]\npath = \"/etc/x/y/a.conf\"\nmode = 0o640\ntemplate = \"a.tmpl\"\n",
		map[string]string{"a.tmpl": "user={{.User}}\n"})
	if err := run(t, e, units); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/etc/x/y/a.conf"]); got != "user=u\n" {
		t.Errorf("content = %q, want user=u", got)
	}
	if got, err := m.RootAttrs("/etc/x/y/a.conf"); err != nil || got != (effect.Attrs{Mode: 0o640}) {
		t.Errorf("attrs = %+v, %v; want root's at 640", got, err)
	}
	if m.Dirs["/etc/x"] != 0o755 || m.Dirs["/etc/x/y"] != 0o755 {
		t.Errorf("parents = %o, %o; want 755", m.Dirs["/etc/x"], m.Dirs["/etc/x/y"])
	}
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
}

func TestFileRefusalLeavesTheLiveFile(t *testing.T) {
	e, m, _, _ := testEnv(false)
	m.Files["/etc/a.conf"] = []byte("old\n")
	m.Results["checker -t /etc/a.conf.ws-check"] = effecttest.Result{Err: errors.New("exit status 1")}
	units := loadOne(t, e, "[file.a]\npath = \"/etc/a.conf\"\nsource = \"a\"\ncheck = [\"checker\", \"-t\", \"{}\"]\n",
		map[string]string{"a": "new\n"})
	err := run(t, e, units)
	if err == nil || !strings.Contains(err.Error(), "checker refuses the new /etc/a.conf") {
		t.Errorf("run = %v, want the check's refusal", err)
	}
	if got := string(m.Files["/etc/a.conf"]); got != "old\n" {
		t.Errorf("live file = %q, want the old one", got)
	}
	if _, left := m.Files["/etc/a.conf.ws-check"]; left {
		t.Error("the checked file stayed beside the live one")
	}
}

func TestFileRefusalOwesTheRestart(t *testing.T) {
	e, m, u, _ := testEnv(true)
	u.States = map[string]effect.UnitState{"a.service": {Active: true}}
	m.Files["/etc/a.conf"] = []byte("old\n")
	m.Results["checker /etc/a.conf.ws-check"] = effecttest.Result{Err: errors.New("exit status 1")}
	const spec = "[file.a]\npath = \"/etc/a.conf\"\nsource = \"a\"\ncheck = [\"checker\", \"{}\"]\nservice = \"a.service\"\n"
	units := loadOne(t, e, spec, map[string]string{"a": "new\n"})
	for range 2 {
		if err := run(t, e, units); err == nil {
			t.Fatal("run succeeded, want the check's refusal")
		}
	}
	if u.Calls != nil || string(m.Files["/etc/a.conf"]) != "old\n" {
		t.Errorf("calls = %q, file %q; want no restart of the old file", u.Calls, m.Files["/etc/a.conf"])
	}
	m.Results["checker /etc/a.conf.ws-check"] = effecttest.Result{}
	if err := run(t, e, units); err != nil || !slices.Equal(u.Calls, []string{"restart a.service"}) {
		t.Errorf("run = %v, calls %q; want the owed restart once the file is good", err, u.Calls)
	}
}

func TestFileCheckBootedSkipsTheCheckUnbooted(t *testing.T) {
	spec := "[file.a]\npath = \"/etc/a.conf\"\nsource = \"a\"\ncheck = [\"checker\", \"{}\"]\ncheck_booted = true\n"
	for _, booted := range []bool{false, true} {
		e, m, _, _ := testEnv(booted)
		m.Results["checker /etc/a.conf.ws-check"] = effecttest.Result{}
		if err := run(t, e, loadOne(t, e, spec, map[string]string{"a": "x"})); err != nil {
			t.Fatal(err)
		}
		if ran := slices.Contains(m.Calls, "root checker /etc/a.conf.ws-check"); ran != booted {
			t.Errorf("booted %t: the check ran %t", booted, ran)
		}
	}
}

// failingRestart is systemd whose restarts fail.
type failingRestart struct{ *effecttest.Units }

func (failingRestart) Restart(context.Context, string) error { return errors.New("boom") }

func TestFileRestartIsOwedAfterAFailedRun(t *testing.T) {
	e, m, u, _ := testEnv(true)
	u.States = map[string]effect.UnitState{"a.service": {Enabled: true, Active: true}}
	e.U = failingRestart{u}
	units := loadOne(t, e, "[file.a]\npath = \"/etc/a.conf\"\nsource = \"a\"\nservice = \"a.service\"\nenabled = true\n",
		map[string]string{"a": "x"})
	if err := run(t, e, units); err == nil {
		t.Fatal("run succeeded, want the restart's failure")
	}
	if string(m.Files["/etc/a.conf"]) != "x" {
		t.Fatal("the file was not written")
	}
	e.U = u
	units = loadOne(t, e, "[file.a]\npath = \"/etc/a.conf\"\nsource = \"a\"\nservice = \"a.service\"\nenabled = true\n",
		map[string]string{"a": "x"})
	if got := headlines(t, units[0].Step); !slices.Equal(got, []string{"restart a.service"}) {
		t.Errorf("plan after the failed run = %q, want the owed restart", got)
	}
	if err := run(t, e, units); err != nil || !slices.Contains(u.Calls, "restart a.service") {
		t.Errorf("run = %v, calls %q; want the restart", err, u.Calls)
	}
}

func TestFileServiceSkipsWithoutSystemd(t *testing.T) {
	e, _, u, out := testEnv(false)
	units := loadOne(t, e, "[file.a]\npath = \"/etc/a.conf\"\nsource = \"a\"\nservice = \"a.service\"\n", map[string]string{"a": "x"})
	if got := headlines(t, units[0].Step); !slices.Equal(got, []string{"create directory /etc, owner 0:0, mode 755", "create /etc/a.conf"}) {
		t.Errorf("plan = %q, want the file alone", got)
	}
	if want := []string{"say: a.service: skipped, systemd is not running"}; !slices.Equal(out.Lines, want) {
		t.Errorf("lines = %q, want %q", out.Lines, want)
	}
	if u.Calls != nil {
		t.Errorf("calls = %q, want none", u.Calls)
	}
}

func TestFileUnderSystemdSystemReloadsTheDaemon(t *testing.T) {
	e, m, u, _ := testEnv(true)
	m.Dirs["/etc/systemd/system"] = 0o755
	units := loadOne(t, e, "[file.a]\npath = \"/etc/systemd/system/a.service\"\nsource = \"a\"\n", map[string]string{"a": "[Unit]\n"})
	if err := run(t, e, units); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(u.Calls, []string{"daemon-reload"}) {
		t.Errorf("calls = %q, want one daemon-reload", u.Calls)
	}
}

func TestFileOfAnotherOwnerStaysInItsHome(t *testing.T) {
	e, m, _, _ := testEnv(false)
	withAccount(m)
	units := loadOne(t, e, "[file.a]\npath = \"/home/u/.config/a/a.conf\"\nowner = \"u\"\nsource = \"a\"\n", map[string]string{"a": "x"})
	if err := run(t, e, units); err != nil {
		t.Fatal(err)
	}
	if got := m.Owners["/home/u/.config/a/a.conf"]; got != [2]int{1001, 1001} || string(m.Files["/home/u/.config/a/a.conf"]) != "x" {
		t.Errorf("file owner = %v, want u's", got)
	}
	if got := m.Owners["/home/u/.config"]; got != [2]int{1001, 1001} {
		t.Errorf("parent owner = %v, want u's", got)
	}
	outside := loadOne(t, e, "[file.a]\npath = \"/etc/a.conf\"\nowner = \"u\"\nsource = \"a\"\n", map[string]string{"a": "x"})
	if _, err := outside[0].Step.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "not below /home/u") {
		t.Errorf("plan = %v, want a path outside the home refused", err)
	}
	m.Links["/home/u/.ssh"] = "/root/.ssh"
	linked := loadOne(t, e, "[file.a]\npath = \"/home/u/.ssh/config\"\nowner = \"u\"\nsource = \"a\"\n", map[string]string{"a": "x"})
	if _, err := linked[0].Step.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "is a link") {
		t.Errorf("plan = %v, want a link on the way refused", err)
	}
}

func TestFileOfAnotherOwnerOnMacOS(t *testing.T) {
	e, m, _, _ := testEnv(false)
	e.System = MacOS
	m.Results["id -u u"] = effecttest.Result{Out: []byte("501\n")}
	m.Results["id -g u"] = effecttest.Result{Out: []byte("20\n")}
	m.Dirs["/Users/u"] = 0o750
	m.Owners["/Users/u"] = [2]int{501, 20}
	const spec = "[file.a]\npath = \"/Users/u/.ssh/config\"\nowner = \"u\"\nsource = \"a\"\n"
	for out, want := range map[string]string{
		"NFSHomeDirectory: /Users/u\n":    "",
		"NFSHomeDirectory:\n /Users/u\n":  "",
		"NFSHomeDirectory: /\n":           "names no home",
		"No such key: NFSHomeDirectory\n": "names no home",
	} {
		m.Results["dscl . -read /Users/u NFSHomeDirectory"] = effecttest.Result{Out: []byte(out)}
		_, err := loadOne(t, e, spec, map[string]string{"a": "x"})[0].Step.Plan(t.Context())
		if want == "" && err != nil || want != "" && (err == nil || !strings.Contains(err.Error(), want)) {
			t.Errorf("dscl gives %q: plan = %v, want %q", out, err, want)
		}
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, "getent") }) {
		t.Errorf("calls = %q, want no getent on macOS", m.Calls)
	}
}

func TestFileAbsentRemoves(t *testing.T) {
	e, m, _, _ := testEnv(false)
	m.Files["/etc/a.conf"] = []byte("x")
	units := loadOne(t, e, "[file.a]\npath = \"/etc/a.conf\"\nabsent = true\n", nil)
	if got := headlines(t, units[0].Step); !slices.Equal(got, []string{"remove /etc/a.conf"}) {
		t.Errorf("plan = %q, want the removal alone", got)
	}
	if err := run(t, e, units); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Files["/etc/a.conf"]; ok {
		t.Error("the file stayed")
	}
}

func TestFileRefusesFieldsNoPlanTakes(t *testing.T) {
	e, _, _, _ := testEnv(false)
	for spec, want := range map[string]string{
		"path = \"/a\"\nabsent = true\ntemplate = \"f\"\n":     "an absent file takes a path, an owner and ours alone",
		"path = \"a\"\nsource = \"f\"\n":                       "not a clean absolute path",
		"path = \"/a/../b\"\nsource = \"f\"\n":                 "not a clean absolute path",
		"path = \"/a\"\n":                                      "a template or a source",
		"path = \"/a\"\nsource = \"f\"\ntemplate = \"f\"\n":    "a template or a source",
		"path = \"/a\"\nsource = \"none\"\n":                   "none",
		"path = \"/a\"\nsource = \"f\"\nmode = 0o4755\n":       "permission bits",
		"path = \"/a\"\nsource = \"f\"\ncheck_booted = true\n": "check_booted goes with check",
		"path = \"/a\"\nsource = \"f\"\nservice = \"a\"\n":     "type's suffix",
		"path = \"/a\"\nsource = \"f\"\nenabled = true\n":      "go with service",
	} {
		_, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": "[file.a]\n" + spec, "r/f": ""}))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("load of\n%s= %v, want %q", spec, err, want)
		}
	}
}

func TestFileSecretKeepsTheDiffOut(t *testing.T) {
	e, m, _, _ := testEnv(false)
	m.Dirs["/etc"] = 0o755
	m.Files["/etc/a.env"] = []byte("TOKEN=old\n")
	units := loadOne(t, e, "[file.a]\npath = \"/etc/a.env\"\nsource = \"a\"\nsecret = true\n", map[string]string{"a": "TOKEN=new\n"})
	p, err := units[0].Step.Plan(t.Context())
	if err != nil || len(p.Changes) != 1 {
		t.Fatalf("plan = %v, %v; want one change", p.Changes, err)
	}
	if got := p.Changes[0].String(); strings.Contains(got, "TOKEN") {
		t.Errorf("change = %q, want no content", got)
	}
}

func TestTemplateRendersAtPlan(t *testing.T) {
	e, m, _, _ := testEnv(false)
	m.Dirs["/etc"] = 0o755
	m.Files["/etc/a"] = []byte("x")
	units := loadOne(t, e, "[file.a]\npath = \"/etc/a\"\ntemplate = \"a.tmpl\"\n", map[string]string{"a.tmpl": "mem={{mem_mib}}\n"})
	if _, err := units[0].Step.Plan(t.Context()); err == nil {
		t.Fatal("plan without the fact succeeded")
	}
	m.Files["/proc/meminfo"] = []byte("MemTotal:        4194304 kB\n")
	if got := headlines(t, units[0].Step); !slices.Equal(got, []string{"write /etc/a"}) {
		t.Errorf("plan = %q, want the write", got)
	}
	if err := run(t, e, units); err != nil || string(m.Files["/etc/a"]) != "mem=4096\n" {
		t.Errorf("run = %v, file %q; want mem=4096", err, m.Files["/etc/a"])
	}
}
