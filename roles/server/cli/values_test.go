package cli

import (
	"strings"
	"testing"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/server/values"
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
	if err := writeVars(m, []byte("---\nWS_USER: ada\n")); err != nil {
		t.Fatal(err)
	}
	if m.Dirs["/etc/ws"] != 0o700 || m.Modes[serverVars] != 0o640 {
		t.Errorf("directory %o, file %o", m.Dirs["/etc/ws"], m.Modes[serverVars])
	}
}

// valuesProcess returns a root process on a machine that holds file as the values.
func valuesProcess(file string) (enginecli.Env, *effecttest.Machine) {
	p := fakeProcess()
	m := effecttest.NewMachine()
	p.Machine, p.Euid = m, 0
	m.Dirs["/etc/ws"] = 0o755
	m.Files[serverVars] = []byte(file)
	return p, m
}

func TestValuesSetWritesOneKey(t *testing.T) {
	p, m := valuesProcess("---\n# the machine\nWS_USER: ada\n")
	status, stdout, stderr := runWs(p, "", "server", "values", "set", "WS_BASE_SWAPPINESS", "20")
	if status != 0 || stdout != "" || !strings.Contains(stderr, "wrote "+serverVars) {
		t.Fatalf("status = %d, stdout %q, stderr %q", status, stdout, stderr)
	}
	if got := string(m.Files[serverVars]); got != "---\n# the machine\nWS_USER: ada\nWS_BASE_SWAPPINESS: 20\n" {
		t.Errorf("file = %q", got)
	}
}

// A seeded file names the home and the signing key, which follow a new user where they hold the
// old one's.
func TestValuesSetMovesTheKeysThatFollow(t *testing.T) {
	v, err := values.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	seeded, err := v.YAML()
	if err != nil {
		t.Fatal(err)
	}
	p, m := valuesProcess(seeded)
	status, _, stderr := runWs(p, "", "server", "values", "set", "WS_USER", "root")
	if status != 0 || !strings.Contains(stderr, "moved WS_HOME and WS_GIT_SIGNING_PUB_KEY to follow WS_USER") {
		t.Fatalf("status = %d, stderr %q", status, stderr)
	}
	got, err := values.Parse(m.Files[serverVars])
	if err != nil || got.Home != "/root" || got.GitSigningPubKey != "/root/.ssh/id_ed25519_signing.pub" {
		t.Errorf("home %q, key %q, %v", got.Home, got.GitSigningPubKey, err)
	}
}

func TestValuesSetAgainWritesNothing(t *testing.T) {
	p, m := valuesProcess("---\nWS_USER: ada\n")
	status, _, stderr := runWs(p, "", "server", "values", "set", "WS_USER", "ada")
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
	p, m := valuesProcess("---\nWS_USER: ada\nWS_HOME: /h\n")
	if status, _, stderr := runWs(p, "", "server", "values", "set", "WS_HOME", ""); status != 0 {
		t.Fatalf("status = %d: %s", status, stderr)
	}
	if got := string(m.Files[serverVars]); got != "---\nWS_USER: ada\n" {
		t.Errorf("file = %q", got)
	}
}

func TestValuesSetTakesAValueWithADash(t *testing.T) {
	p, m := valuesProcess("---\n")
	if status, _, stderr := runWs(p, "", "server", "values", "set", "--", "WS_USER", "-x"); status != 0 {
		t.Fatalf("status = %d: %s", status, stderr)
	}
	if got := string(m.Files[serverVars]); got != "---\nWS_USER: -x\n" {
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
		p    enginecli.Env
		args []string
		want string
	}{
		{"not root", notRoot, []string{"WS_USER", "ada"}, "run it as root"},
		{"a link", linked, []string{"WS_USER", "ada"}, "is a link to /srv/vars.yml"},
		{"a table", plain, []string{"WS_FANS_CURVES", "1"}, "no line holds WS_FANS_CURVES"},
		{"a wrong kind", plain, []string{"WS_BASE_SWAPPINESS", "ten"}, "a whole number"},
		{"no value", plain, []string{"WS_USER"}, "accepts 2 arg"},
	} {
		args := append([]string{"server", "values", "set"}, tc.args...)
		if status, _, stderr := runWs(tc.p, "", args...); status == 0 || !strings.Contains(stderr, tc.want) {
			t.Errorf("%s: status = %d, stderr %q, want %q", tc.name, status, stderr, tc.want)
		}
	}
}
