package cmd

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestWriteVarsStartsTheFile(t *testing.T) {
	m := effecttest.NewMachine()
	if err := writeVars(m, []byte("---\n")); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files[serverVars]); got != "---\n" {
		t.Errorf("file = %q", got)
	}
	if m.Dirs["/etc/ws"] != 0o755 || m.Modes[serverVars] != 0o600 {
		t.Errorf("directory %o, file %o", m.Dirs["/etc/ws"], m.Modes[serverVars])
	}
}

func TestWriteVarsKeepsTheModeOfAFile(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/etc/ws"] = 0o700
	m.Files[serverVars] = []byte("---\n")
	m.Modes[serverVars] = 0o640
	if err := writeVars(m, []byte("---\nws_user: ada\n")); err != nil {
		t.Fatal(err)
	}
	if m.Dirs["/etc/ws"] != 0o700 || m.Modes[serverVars] != 0o640 {
		t.Errorf("directory %o, file %o", m.Dirs["/etc/ws"], m.Modes[serverVars])
	}
}

// valuesProcess returns a root process on a machine that holds file as the values.
func valuesProcess(file string) (cli.Env, *effecttest.Machine) {
	p := fakeProcess()
	m := effecttest.NewMachine()
	p.Machine, p.Euid = m, 0
	m.Dirs["/etc/ws"] = 0o755
	m.Files[serverVars] = []byte(file)
	return p, m
}

func TestValuesSetWritesOneKey(t *testing.T) {
	p, m := valuesProcess("---\n# the machine\nws_user: ada\n")
	status, stdout, stderr := runWs(p, "", "server", "ubuntu", "values", "set", "base_swappiness", "20")
	if status != 0 || stdout != "" || !strings.Contains(stderr, "wrote "+serverVars) {
		t.Fatalf("status = %d, stdout %q, stderr %q", status, stdout, stderr)
	}
	if got := string(m.Files[serverVars]); got != "---\n# the machine\nws_user: ada\nbase_swappiness: 20\n" {
		t.Errorf("file = %q", got)
	}
}

func TestValuesSetAgainWritesNothing(t *testing.T) {
	p, m := valuesProcess("---\nws_user: ada\n")
	status, _, stderr := runWs(p, "", "server", "ubuntu", "values", "set", "ws_user", "ada")
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
	p, m := valuesProcess("---\nws_user: ada\nws_home: /h\n")
	if status, _, stderr := runWs(p, "", "server", "ubuntu", "values", "set", "ws_home", ""); status != 0 {
		t.Fatalf("status = %d: %s", status, stderr)
	}
	if got := string(m.Files[serverVars]); got != "---\nws_user: ada\n" {
		t.Errorf("file = %q", got)
	}
}

func TestValuesSetTakesAValueWithADash(t *testing.T) {
	p, m := valuesProcess("---\n")
	if status, _, stderr := runWs(p, "", "server", "ubuntu", "values", "set", "--", "ws_user", "-x"); status != 0 {
		t.Fatalf("status = %d: %s", status, stderr)
	}
	if got := string(m.Files[serverVars]); got != "---\nws_user: -x\n" {
		t.Errorf("file = %q", got)
	}
}

func TestValuesSetRefuses(t *testing.T) {
	notRoot, _ := valuesProcess("---\n")
	notRoot.Euid = 501
	linked, m := valuesProcess("---\n")
	m.Links[serverVars] = "/srv/vars.yml"
	plain, _ := valuesProcess("---\n")
	for _, tc := range []struct {
		name string
		p    cli.Env
		args []string
		want string
	}{
		{"not root", notRoot, []string{"ws_user", "ada"}, "run it as root"},
		{"a link", linked, []string{"ws_user", "ada"}, "is a link to /srv/vars.yml"},
		{"a table", plain, []string{"fans_curves", "1"}, "no line holds fans_curves"},
		{"a wrong kind", plain, []string{"base_swappiness", "ten"}, "a whole number"},
		{"no value", plain, []string{"ws_user"}, "accepts 2 arg"},
	} {
		args := append([]string{"server", "ubuntu", "values", "set"}, tc.args...)
		if status, _, stderr := runWs(tc.p, "", args...); status == 0 || !strings.Contains(stderr, tc.want) {
			t.Errorf("%s: status = %d, stderr %q, want %q", tc.name, status, stderr, tc.want)
		}
	}
}
