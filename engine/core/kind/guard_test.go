package kind

import (
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
		"after = \"\"\n": "takes login_key with home, or sudo_right",
		"login_key = \"u\"\nhome = \"/home/u\"\nsudo_right = \"u\"\n": "takes login_key with home, or sudo_right",
		"sudo_right = \"u\"\nhome = \"/home/u\"\n":                    "takes login_key with home, or sudo_right",
		"login_key = \"u\"\n":                                         `home "" is not a clean absolute path`,
		"login_key = \"u\"\nhome = \"home/u\"\n":                      `home "home/u" is not a clean absolute path`,
		"sudo_right = \"-u\"\n":                                       `sudo_right "-u" is no user's name`,
	} {
		_, err := Load(e, tree(map[string]string{"r/units.toml": "[guard.a]\n" + spec}))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("load of\n%s= %v, want %q", spec, err, want)
		}
	}
}
