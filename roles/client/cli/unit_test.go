package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/cli/clitest"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/values"
)

// runOfTheUser is a spec whose run state is the Mac's user's, with a check that passes.
const runOfTheUser = "[state.mine.run]\nuser = \"{{ .User }}\"\ncommand = [\"true\"]\ncheck = [\"test\", \"-d\", \"{{ .Home }}\"]\n"

// macUnitEnv returns the user v's Env, whose WS_UNITS holds the spec, and its machine, with no
// server and a file key.
func macUnitEnv(t *testing.T, spec string) (enginecli.Env, *effecttest.Machine) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "r"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "r", "units.toml"), []byte(spec), 0o600); err != nil {
		t.Fatal(err)
	}
	m := effecttest.NewMachine()
	m.Files["/etc/ws/values.yml"] = []byte(values.Text("WS_IDENTITY_PUB_KEY", "~/.ssh/id.pub",
		"WS_GIT_SIGNING_PUB_KEY", "~/.ssh/id.pub"))
	m.Files["/Users/v/.ssh/id.pub"] = []byte("ssh-ed25519 AAAA me\n")
	m.Results["ssh-keygen -l -f /Users/v/.ssh/id.pub"] = effecttest.Result{}
	p := withEnv(fakeProcess(), map[string]string{"WS_UNITS": dir, "USER": "v", "HOME": "/Users/v"})
	p.Machine = m
	return p, m
}

// runUnit runs ws unit, the Mac's, with args.
func runUnit(p enginecli.Env, args ...string) (status int, stdout, stderr string) {
	return clitest.Run(p, []*cobra.Command{UnitCmd(p)}, "", append([]string{"unit"}, args...)...)
}

func TestMacUnitSide(t *testing.T) {
	s := unitSide(fakeProcess())
	if s.Dir != "/opt/ws/roles/client/macos" || s.Root || s.Units != nil || s.Secrets != nil || s.System.String() != "macOS" {
		t.Errorf("side = %+v, want the Mac's specs, no root, no systemd, no secrets", s)
	}
	p, _ := macUnitEnv(t, runOfTheUser)
	if status, stdout, stderr := runUnit(p, "list", "--toml"); status != 0 || stdout != "\"ws:state/mine\" = \"in-sync\"\n" {
		t.Errorf("list: status = %d, stdout = %q, stderr = %q; want the spec of WS_UNITS", status, stdout, stderr)
	}
}

func TestMacUnitReadsAsTheUser(t *testing.T) {
	p, m := macUnitEnv(t, runOfTheUser)
	m.Results["env PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin test -d /Users/v"] = effecttest.Result{}
	if status, stdout, stderr := runUnit(p, "read", "state/mine"); status != 0 || stdout != "state/mine\tin-sync\n" {
		t.Errorf("read: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	for _, c := range m.Calls {
		if strings.HasPrefix(c, "root ") {
			t.Errorf("read called %q, want no sudo", c)
		}
	}
	if !slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasSuffix(c, "test -d /Users/v") }) {
		t.Errorf("calls = %q, want the check", m.Calls)
	}
}

func TestMacUnitRefusesRoot(t *testing.T) {
	p, _ := macUnitEnv(t, runOfTheUser)
	p.Euid = 0
	if status, _, stderr := runUnit(p, "list"); status != 1 || !strings.Contains(stderr, "not root") {
		t.Errorf("list as root: status = %d, stderr = %q; want the refusal", status, stderr)
	}
}

func TestMacUnitRunsTheUsersCommandBare(t *testing.T) {
	p, m := macUnitEnv(t, strings.Replace(runOfTheUser, "-d", "-f", 1))
	m.Results["env PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin true"] = effecttest.Result{}
	status, _, stderr := runUnit(p, "write", "state/mine")
	if !slices.Contains(m.Calls, "run env PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin true") {
		t.Errorf("write: status = %d, stderr = %q, calls %q; want the command run as v with no sudo", status, stderr, m.Calls)
	}
}
