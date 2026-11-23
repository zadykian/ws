package kind

// The kinds on the Mac, where ws runs as the user: a run's check as that user, and root's files
// readable by all.

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// macRun loads a run state on the Mac, where ws runs as the user v, whose command fails its check.
func macRun(t *testing.T, user string) (*effecttest.Machine, []Unit, *effecttest.Out) {
	e, m, _, out := testEnv()
	e.System, e.User = MacOS, "v"
	m.Results["env "+systemPath+" cld --check"] = effecttest.Result{Err: errors.New("exit status 1")}
	return m, loadState(t, e, "run = { user = \""+user+"\", command = [\"cld\", \"--install\"], check = [\"cld\", \"--check\"] }\n"), out
}

func TestStateRunOnMacOSChecksAsTheUser(t *testing.T) {
	for _, user := range []string{"v", "root", "u"} {
		m, units, out := macRun(t, user)
		if got := headlines(t, units[0].Step); !slices.Equal(got, []string{"run cld --install as " + user}) || out.Lines != nil {
			t.Errorf("%s: plan = %q, lines %q; want the run, with no systemd", user, got, out.Lines)
		}
		if want := []string{"run env " + systemPath + " cld --check"}; !slices.Equal(m.Calls, want) {
			t.Errorf("%s: the plan's calls = %q, want the check as ws's user alone, %q", user, m.Calls, want)
		}
	}
}

func TestStateRunOnMacOSCommand(t *testing.T) {
	const line = "env " + systemPath + " cld --install"
	for user, want := range map[string]string{
		"v":    "run " + line,
		"root": "root " + line,
		"u":    "root sudo -u u -H " + line,
	} {
		m, units, _ := macRun(t, user)
		_, cmd, _ := strings.Cut(want, " ")
		m.Results[cmd] = effecttest.Result{}
		if err := applyOnce(t, units[0].Step); err != nil || !slices.Contains(m.Calls, want) {
			t.Errorf("%s: apply = %v, calls %q; want %q", user, err, m.Calls, want)
		}
	}
}

func TestStateRunOnMacOSPutsItsPathFirst(t *testing.T) {
	e, m, _, _ := testEnv()
	e.System, e.User = MacOS, "v"
	const env = "env PATH=/Users/v/.local/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin "
	m.Results[env+"cld --check"] = effecttest.Result{Err: errors.New("exit status 1")}
	m.Results[env+"cld --install"] = effecttest.Result{}
	units := loadState(t, e, "run = { user = \"v\", path = [\"/Users/v/.local/bin\"], "+
		"command = [\"cld\", \"--install\"], check = [\"cld\", \"--check\"] }\n")
	if err := applyOnce(t, units[0].Step); err != nil || !slices.Contains(m.Calls, "run "+env+"cld --install") {
		t.Errorf("apply = %v, calls %q; want the check and the command with the path first", err, m.Calls)
	}
}

func TestMacRefusesAnUnreadableRootFile(t *testing.T) {
	e, _, _, _ := testEnv()
	e.System = MacOS
	files := map[string]string{"r/a": "x"}
	for spec, refused := range map[string]bool{
		"path = \"/etc/ws/a\"\nsource = \"a\"\nmode = 0o600\n":                      true,
		"path = \"/etc/ws/a\"\nsource = \"a\"\nmode = 0o640\nowner = \"root\"\n":    true,
		"path = \"/etc/ws/a\"\nsource = \"a\"\nmode = 0o644\n":                      false,
		"path = \"/etc/ws/a\"\nsource = \"a\"\n":                                    false,
		"path = \"/etc/ws/a\"\nabsent = true\n":                                     false,
		"path = \"/Users/u/.ssh/a\"\nsource = \"a\"\nmode = 0o600\nowner = \"u\"\n": false,
	} {
		files["r/units.toml.tmpl"] = "[file.a]\n" + spec
		_, err := Load(e, tree(files))
		if got := err != nil && strings.Contains(err.Error(), "readable by all"); got != refused {
			t.Errorf("%q: load = %v, want refused %v", spec, err, refused)
		}
	}
	e.System = Linux
	files["r/units.toml.tmpl"] = "[file.a]\npath = \"/etc/ws/a\"\nsource = \"a\"\nmode = 0o600\n"
	if _, err := Load(e, tree(files)); err != nil {
		t.Errorf("load of a 0600 root file on Linux = %v, want none", err)
	}
}
