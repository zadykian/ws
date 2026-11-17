package kind

import (
	"errors"
	"io/fs"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

const (
	keysOfU    = "/home/u/.ssh/authorized_keys"
	loginGuard = "[guard.a]\nlogin_key = \"u\"\nhome = \"/home/u\"\n"
	sudoGuard  = "[guard.a]\nsudo_right = \"u\"\n"
)

// guardPlan loads the guard of spec and plans it, which returns no change or the lockout's error.
func guardPlan(t *testing.T, e Env, spec string) error {
	units := loadOne(t, e, spec, nil)
	p, err := units[0].Step.Plan(t.Context())
	if err == nil && len(p.Changes) > 0 {
		t.Errorf("plan = %q, want none: a guard plans nothing", p.Changes)
	}
	return err
}

// withKeys gives u an authorized_keys of keys, u's own at mode.
func withKeys(m *effecttest.Machine, keys string, mode fs.FileMode) {
	m.Dirs["/home/u/.ssh"] = 0o700
	m.Owners["/home/u/.ssh"] = [2]int{1001, 1001}
	m.Files[keysOfU] = []byte(keys)
	m.Owners[keysOfU] = [2]int{1001, 1001}
	m.Modes[keysOfU] = mode
}

func TestGuardLoginKeyPasses(t *testing.T) {
	e, m, _, _ := testEnv(true)
	withAccount(m)
	withKeys(m, "# the Mac\nssh-ed25519 AAAA u@mac\n", 0o600)
	if err := guardPlan(t, e, loginGuard); err != nil {
		t.Errorf("plan = %v, want none", err)
	}
	reports, err := step.Read(t.Context(), Steps(loadOne(t, e, loginGuard, nil)))
	if err != nil || len(reports) != 1 || reports[0].Changes != nil || slices.ContainsFunc(reports[0].Items,
		func(it step.Item) bool { return it.Kind == step.CheckItem }) {
		t.Errorf("read = %+v, %v; want the guard in sync", reports, err)
	}
}

func TestGuardRefusesNoLoginKey(t *testing.T) {
	for name, c := range map[string]struct {
		keys string
		mode fs.FileMode
		link bool
		want string
	}{
		"missing":  {want: keysOfU + " holds no key"},
		"comments": {keys: "# none\n\n", mode: 0o600, want: keysOfU + " holds no key"},
		"writable": {keys: "ssh-ed25519 AAAA\n", mode: 0o666, want: "sshd would pass over " + keysOfU},
		"linked":   {keys: "ssh-ed25519 AAAA\n", mode: 0o600, link: true, want: "link"},
	} {
		e, m, _, _ := testEnv(true)
		withAccount(m)
		if c.keys != "" {
			withKeys(m, c.keys, c.mode)
		}
		if c.link {
			m.Links["/home/u/.ssh"] = "/srv/keys"
		}
		if err := guardPlan(t, e, loginGuard); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: plan = %v, want %q", name, err, c.want)
		}
	}
}

func TestGuardSudoRight(t *testing.T) {
	const may = "User u may run the following commands on ws:\n    (ALL : ALL) "
	for name, c := range map[string]struct {
		list, status string
		want         string
	}{
		"password":    {may + "ALL\n", "u P 2026-11-17 0 99999 7 -1\n", ""},
		"no password": {may + "NOPASSWD: ALL\n", "", ""},
		"no right":    {"User u is not allowed to run sudo on ws.\n", "", "u has no right to sudo"},
		"locked":      {may + "ALL\n", "u L 2026-11-17 0 99999 7 -1\n", "u has no password for sudo to ask"},
	} {
		e, m, _, _ := testEnv(true)
		m.Results["sudo -l -U u"] = effecttest.Result{Out: []byte(c.list)}
		m.Results["passwd --status u"] = effecttest.Result{Out: []byte(c.status)}
		err := guardPlan(t, e, sudoGuard)
		if (c.want == "" && err != nil) || (c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want))) {
			t.Errorf("%s: plan = %v, want %q", name, err, c.want)
		}
	}
}

func TestGuardSudoRightPassesRoot(t *testing.T) {
	e, m, _, _ := testEnv(true)
	if err := guardPlan(t, e, "[guard.a]\nsudo_right = \"root\"\n"); err != nil || slices.ContainsFunc(m.Calls,
		func(c string) bool { return strings.HasPrefix(c, "root sudo") }) {
		t.Errorf("plan = %v, calls %q; want none, and no question to sudo", err, m.Calls)
	}
}

func TestGuardPassesInAContainer(t *testing.T) {
	e, m, _, _ := testEnv(true)
	withAccount(m)
	m.Files["/.dockerenv"] = nil
	for _, spec := range []string{loginGuard, sudoGuard} {
		if err := guardPlan(t, e, spec); err != nil {
			t.Errorf("plan of\n%s= %v, want none in a container", spec, err)
		}
	}
}

func TestGuardTakesExactlyOne(t *testing.T) {
	e, _, _, _ := testEnv(false)
	for spec, want := range map[string]string{
		"after = \"\"\n": "takes one of login_key with home, sudo_right and command",
		"login_key = \"u\"\nhome = \"/home/u\"\nsudo_right = \"u\"\n":        "takes one of login_key with home, sudo_right and command",
		"sudo_right = \"u\"\nhome = \"/home/u\"\n":                           "takes one of login_key with home, sudo_right and command",
		"login_key = \"u\"\n":                                                `home "" is not a clean absolute path`,
		"login_key = \"u\"\nhome = \"home/u\"\n":                             `home "home/u" is not a clean absolute path`,
		"sudo_right = \"-u\"\n":                                              `sudo_right "-u" is no user's name`,
		"command = [\"/bin/true\"]\nlogin_key = \"u\"\nhome = \"/home/u\"\n": "takes one of login_key with home, sudo_right and command",
		"command = [\"/bin/true\"]\nsudo_right = \"u\"\n":                    "takes one of login_key with home, sudo_right and command",
		"command = [\"/bin/true\"]\nhome = \"/home/u\"\n":                    "takes one of login_key with home, sudo_right and command",
		"command = [\"true\"]\n":                                             `command "true": its program is no absolute path`,
		"command = [\"\"]\n":                                                 `command "": its program is no absolute path`,
	} {
		_, err := Load(e, tree(map[string]string{"r/units.toml": "[guard.a]\n" + spec}))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("load of\n%s= %v, want %q", spec, err, want)
		}
	}
}

const commandGuard = "[guard.a]\ncommand = [\"/usr/bin/check\", \"x\"]\n"

func TestGuardCommandPassesAndSays(t *testing.T) {
	e, m, _, out := testEnv(true)
	m.Results["/usr/bin/check x"] = effecttest.Result{Out: []byte("first\n\nsecond\n")}
	units := loadOne(t, e, commandGuard, nil)
	if err := run(t, e, units); err != nil {
		t.Fatalf("run = %v, want none", err)
	}
	if !slices.Contains(m.Calls, "root /usr/bin/check x") {
		t.Errorf("calls %q, want the command as root", m.Calls)
	}
	if want := []string{"say: first", "say: second"}; !slices.Equal(out.Lines, want) {
		t.Errorf("lines = %q, want %q once, though the run plans twice", out.Lines, want)
	}
}

func TestGuardCommandFailsWithItsOutput(t *testing.T) {
	e, m, _, _ := testEnv(true)
	m.Results["/usr/bin/check x"] = effecttest.Result{Out: []byte("the port is an uplink\n"), Err: errors.New("exit status 1")}
	err := guardPlan(t, e, commandGuard)
	if err == nil || !strings.Contains(err.Error(), "exit status 1") || !strings.Contains(err.Error(), "the port is an uplink") {
		t.Errorf("plan = %v, want the command's error and output", err)
	}
}

func TestGuardCommandRunsInAContainer(t *testing.T) {
	e, m, _, _ := testEnv(false)
	m.Files["/.dockerenv"] = nil
	m.Results["/usr/bin/check x"] = effecttest.Result{Err: errors.New("exit status 1")}
	if err := guardPlan(t, e, commandGuard); err == nil {
		t.Error("plan = nil, want the command's refusal in a container too")
	}
}
