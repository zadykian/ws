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
	e, m, _, _ := testEnv()
	withAccount(m)
	withKeys(m, "# the Mac\nssh-ed25519 AAAA u@mac\n", 0o600)
	if err := guardPlan(t, e, loginGuard); err != nil {
		t.Errorf("plan = %v, want none", err)
	}
	reports, err := step.Read(t.Context(), Steps(loadOne(t, e, loginGuard, nil)))
	if err != nil || len(reports) != 1 || reports[0].Changes != nil || reports[0].Err != nil {
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
		e, m, _, _ := testEnv()
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

func TestGuardLoginKeyCounts(t *testing.T) {
	cloudInit := `no-port-forwarding,no-agent-forwarding,no-X11-forwarding,command="echo 'Please login as the user \"ubuntu\" rather than the user \"root\".';echo;sleep 10;exit 142" ssh-ed25519 AAAA`
	for name, c := range map[string]struct{ keys, want string }{
		"cloud-init alone":   {keys: cloudInit + "\n", want: "behind a forced command"},
		"a plain key":        {keys: "ssh-ed25519 AAAA u@mac\n"},
		"another option":     {keys: `from="10.0.0.0/8" ssh-ed25519 AAAA` + "\n"},
		"a mix":              {keys: cloudInit + "\nssh-ed25519 AAAA u@mac\n"},
		"a comma in a quote": {keys: `from="a,b",command="x" ssh-ed25519 AAAA` + "\n", want: "behind a forced command"},
		"command in a value": {keys: `from="command=x" ssh-ed25519 AAAA` + "\n"},
		"upper case":         {keys: `COMMAND="x" ssh-rsa AAAA` + "\n", want: "behind a forced command"},
		"no key after":       {keys: "no-pty\n", want: "holds no key"},
		"a quote not closed": {keys: `from="a ssh-ed25519 AAAA` + "\n", want: "holds no key"},
		"a certificate":      {keys: "ssh-ed25519-cert-v01@openssh.com AAAA\n"},
		"a hardware key":     {keys: "sk-ssh-ed25519@openssh.com AAAA\n"},
	} {
		e, m, _, _ := testEnv()
		withAccount(m)
		withKeys(m, c.keys, 0o600)
		err := guardPlan(t, e, loginGuard)
		if c.want == "" && err != nil || c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)) {
			t.Errorf("%s: plan = %v, want %q", name, err, c.want)
		}
	}
}

func TestGuardRefusesAHomeOtherThanTheAccounts(t *testing.T) {
	for _, container := range []bool{false, true} {
		e, m, _, _ := testEnv()
		withAccount(m)
		withKeys(m, "ssh-ed25519 AAAA u@mac\n", 0o600)
		if container {
			m.Files["/.dockerenv"] = nil
		}
		err := guardPlan(t, e, "[guard.a]\nlogin_key = \"u\"\nhome = \"/srv/u\"\n")
		if err == nil || !strings.Contains(err.Error(), "home /srv/u is not u's home, /home/u") {
			t.Errorf("container %t: plan = %v, want the home refused", container, err)
		}
	}
}

func TestGuardRefusesNoLoginKeyInAContainer(t *testing.T) {
	e, m, _, _ := testEnv()
	withAccount(m)
	m.Files["/.dockerenv"] = nil
	if err := guardPlan(t, e, loginGuard); err == nil || !strings.Contains(err.Error(), "holds no key") {
		t.Errorf("plan = %v, want the missing key refused in a container too", err)
	}
}

func TestGuardTakesExactlyOne(t *testing.T) {
	e, _, _, _ := testEnv()
	for spec, want := range map[string]string{
		"after = \"\"\n":                         "takes one of login_key with home and command",
		"login_key = \"u\"\n":                    `home "" is not a clean absolute path`,
		"login_key = \"u\"\nhome = \"home/u\"\n": `home "home/u" is not a clean absolute path`,
		"command = [\"/bin/true\"]\nlogin_key = \"u\"\nhome = \"/home/u\"\n": "takes one of login_key with home and command",
		"command = [\"/bin/true\"]\nhome = \"/home/u\"\n":                    "takes one of login_key with home and command",
		"command = [\"true\"]\n":                                             `command "true": its program is no absolute path`,
		"command = [\"\"]\n":                                                 `command "": its program is no absolute path`,
	} {
		_, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": "[guard.a]\n" + spec}))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("load of\n%s= %v, want %q", spec, err, want)
		}
	}
}

const commandGuard = "[guard.a]\ncommand = [\"/usr/bin/check\", \"x\"]\n"

func TestGuardCommandPassesAndSays(t *testing.T) {
	e, m, _, out := testEnv()
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
	e, m, _, _ := testEnv()
	m.Results["/usr/bin/check x"] = effecttest.Result{Out: []byte("the port is an uplink\n"), Err: errors.New("exit status 1")}
	err := guardPlan(t, e, commandGuard)
	if err == nil || !strings.Contains(err.Error(), "exit status 1") || !strings.Contains(err.Error(), "the port is an uplink") {
		t.Errorf("plan = %v, want the command's error and output", err)
	}
}

func TestGuardCommandRunsInAContainer(t *testing.T) {
	e, m, _, _ := testEnv()
	m.Files["/.dockerenv"] = nil
	m.Results["/usr/bin/check x"] = effecttest.Result{Err: errors.New("exit status 1")}
	if err := guardPlan(t, e, commandGuard); err == nil {
		t.Error("plan = nil, want the command's refusal in a container too")
	}
}
