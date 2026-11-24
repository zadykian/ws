package cli

import (
	"strings"
	"testing"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestWriteVarsStartsTheFile(t *testing.T) {
	m := effecttest.NewMachine()
	if err := writeVars(m, []byte("---\n")); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files[guestVars]); got != "---\n" {
		t.Errorf("file = %q", got)
	}
	if m.Dirs["/etc/ws"] != 0o755 || m.Modes[guestVars] != 0o600 {
		t.Errorf("directory %o, file %o", m.Dirs["/etc/ws"], m.Modes[guestVars])
	}
}

func TestWriteVarsKeepsTheModeOfAFile(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/etc/ws"] = 0o700
	m.Files[guestVars] = []byte("---\n")
	m.Modes[guestVars] = 0o640
	if err := writeVars(m, []byte("---\nWS_BASE_LOCALE: C\n")); err != nil {
		t.Fatal(err)
	}
	if m.Dirs["/etc/ws"] != 0o700 || m.Modes[guestVars] != 0o640 {
		t.Errorf("directory %o, file %o", m.Dirs["/etc/ws"], m.Modes[guestVars])
	}
}

// valuesProcess returns a root process on a machine that holds file as the values.
func valuesProcess(file string) (enginecli.Env, *effecttest.Machine) {
	p := fakeProcess()
	m := effecttest.NewMachine()
	p.Machine, p.Euid = m, 0
	m.Dirs["/etc/ws"] = 0o755
	m.Files[guestVars] = []byte(file)
	return p, m
}

func TestValuesSetWritesOneKey(t *testing.T) {
	p, m := valuesProcess("---\n# the machine\nWS_BASE_LOCALE: C\n")
	status, stdout, stderr := runWs(p, "", "ubuntu", "guest", "values", "set", "WS_BASE_SWAPPINESS", "20")
	if status != 0 || stdout != "" || !strings.Contains(stderr, "wrote "+guestVars) {
		t.Fatalf("status = %d, stdout %q, stderr %q", status, stdout, stderr)
	}
	if got := string(m.Files[guestVars]); got != "---\n# the machine\nWS_BASE_LOCALE: C\nWS_BASE_SWAPPINESS: 20\n" {
		t.Errorf("file = %q", got)
	}
}

func TestValuesSetAgainWritesNothing(t *testing.T) {
	p, m := valuesProcess("---\nWS_BASE_LOCALE: C\n")
	status, _, stderr := runWs(p, "", "ubuntu", "guest", "values", "set", "WS_BASE_LOCALE", "C")
	if status != 0 || strings.Contains(stderr, "wrote") {
		t.Fatalf("status = %d, stderr %q", status, stderr)
	}
	for _, call := range m.Calls {
		if strings.HasPrefix(call, "atomic") {
			t.Errorf("wrote the file: %s", call)
		}
	}
}

func TestValuesSetTakesAKeyOut(t *testing.T) {
	p, m := valuesProcess("---\nWS_BASE_LOCALE: C\nWS_BASE_TIMEZONE: UTC\n")
	if status, _, stderr := runWs(p, "", "ubuntu", "guest", "values", "set", "WS_BASE_TIMEZONE", ""); status != 0 {
		t.Fatalf("status = %d: %s", status, stderr)
	}
	if got := string(m.Files[guestVars]); got != "---\nWS_BASE_LOCALE: C\n" {
		t.Errorf("file = %q", got)
	}
}

func TestValuesSetTakesAValueWithADash(t *testing.T) {
	p, m := valuesProcess("---\n")
	if status, _, stderr := runWs(p, "", "ubuntu", "guest", "values", "set", "--", "WS_BASE_LOCALE", "-x"); status != 0 {
		t.Fatalf("status = %d: %s", status, stderr)
	}
	if got := string(m.Files[guestVars]); got != "---\nWS_BASE_LOCALE: -x\n" {
		t.Errorf("file = %q", got)
	}
}

func TestValuesSetRefuses(t *testing.T) {
	notRoot, _ := valuesProcess("---\n")
	notRoot.Euid = 501
	linked, m := valuesProcess("---\n")
	m.Links[guestVars] = "/srv/vars.yml"
	plain, _ := valuesProcess("---\n")
	for _, tc := range []struct {
		name string
		p    enginecli.Env
		args []string
		want string
	}{
		{"not root", notRoot, []string{"WS_BASE_LOCALE", "C"}, "run it as root"},
		{"a link", linked, []string{"WS_BASE_LOCALE", "C"}, "is a link to /srv/vars.yml"},
		{"a table", plain, []string{"WS_FANS_CURVES", "1"}, "no line holds WS_FANS_CURVES"},
		{"a wrong kind", plain, []string{"WS_BASE_SWAPPINESS", "ten"}, "a whole number"},
		{"no value", plain, []string{"WS_BASE_LOCALE"}, "accepts 2 arg"},
	} {
		args := append([]string{"ubuntu", "guest", "values", "set"}, tc.args...)
		if status, _, stderr := runWs(tc.p, "", args...); status == 0 || !strings.Contains(stderr, tc.want) {
			t.Errorf("%s: status = %d, stderr %q, want %q", tc.name, status, stderr, tc.want)
		}
	}
}
