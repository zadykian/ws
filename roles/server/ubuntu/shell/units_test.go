package shell_test

import (
	"context"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

const envPath = "/etc/ws/env"

// role is the shell's spec and its env template, as kind.Load finds a role's.
func role(t *testing.T) fstest.MapFS {
	fsys := fstest.MapFS{}
	for _, name := range []string{"units.toml", "files/env.tmpl"} {
		text, err := os.ReadFile(name) //nolint:gosec // a file of the role, named above
		if err != nil {
			t.Fatal(err)
		}
		fsys["shell/"+name] = &fstest.MapFile{Data: text}
	}
	return fsys
}

// machine is a machine with the account zadykian, 1000:1000, and /etc/ws, and its env for the
// default values, whose secrets come from load.
func machine(t *testing.T, load func(context.Context) (any, error)) (*effecttest.Machine, kind.Env) {
	v, err := values.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	m := effecttest.NewMachine()
	m.Results["id -u zadykian"] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["id -g zadykian"] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["getent passwd zadykian"] = effecttest.Result{Out: []byte("zadykian:x:1000:1000::/home/zadykian:/bin/bash\n")}
	m.Results["getent passwd root"] = effecttest.Result{Out: []byte("root:x:0:0:root:/root:/bin/bash\n")}
	m.Dirs["/home/zadykian"], m.Owners["/home/zadykian"] = 0o750, [2]int{1000, 1000}
	m.Dirs["/root"], m.Dirs["/etc"], m.Dirs["/etc/ws"] = 0o700, 0o755, 0o755
	return m, kind.Env{M: m, U: &effecttest.Units{}, Out: &effecttest.Out{}, Data: v, Secrets: load}
}

// table returns secrets whose [env] table is env.
func table(env map[string]string) func(context.Context) (any, error) {
	return func(context.Context) (any, error) { return secrets.Secrets{Env: env, Whole: true}, nil }
}

// units loads the shell's units on e, and keeps those named.
func units(t *testing.T, e kind.Env, names ...string) []kind.Unit {
	all, err := kind.Load(e, role(t))
	if err != nil {
		t.Fatal(err)
	}
	return slices.DeleteFunc(all, func(u kind.Unit) bool { return len(names) > 0 && !slices.Contains(names, u.Name) })
}

// write runs the units as ws unit write does.
func write(t *testing.T, e kind.Env, us []kind.Unit) error {
	_, err := step.Run(t.Context(), kind.Steps(us), e.Out, false)
	return err
}

// changes returns the headlines of the unit's plan.
func changes(t *testing.T, u kind.Unit) []string {
	p, err := u.Step.Plan(t.Context())
	if err != nil {
		t.Fatalf("plan %s: %v", u.Name, err)
	}
	var got []string
	for _, c := range p.Changes {
		got = append(got, step.Headline(c))
	}
	return got
}

func TestShellUnitsForAnAccountAndRoot(t *testing.T) {
	m, e := machine(t, nil)
	m.Files["/home/zadykian/.bashrc"] = []byte("# stock\n")
	us := units(t, e)
	want := []string{"file/shell-bashrc-line", "file/shell-env", "file/shell-profile-line"}
	if got := kind.Names(us); !slices.Equal(got, want) {
		t.Fatalf("units = %q, want %q", got, want)
	}
	if err := write(t, e, us); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/home/zadykian/.bashrc"]); got != ". /etc/ws/bashrc\n\n# stock\n" {
		t.Errorf(".bashrc = %q, want the line at the top", got)
	}
	if got := string(m.Files["/home/zadykian/.profile"]); got != ". /etc/ws/profile\n" || m.Owners["/home/zadykian/.profile"] != [2]int{1000, 1000} {
		t.Errorf(".profile = %q, owner %v; want the line, the account's", got, m.Owners["/home/zadykian/.profile"])
	}
	for _, u := range us {
		if got := changes(t, u); got != nil {
			t.Errorf("second plan of %s = %q, want none", u.Name, got)
		}
	}
	root, err := values.Parse([]byte("WS_USER: root\n"))
	if err != nil {
		t.Fatal(err)
	}
	e.Data = root
	if err := write(t, e, units(t, e)); err != nil {
		t.Fatal(err)
	}
	if string(m.Files["/root/.bashrc"]) != ". /etc/ws/bashrc\n" || string(m.Files["/root/.profile"]) != ". /etc/ws/profile\n" {
		t.Errorf("root's files = %q, %q; want the lines", m.Files["/root/.bashrc"], m.Files["/root/.profile"])
	}
}

func TestShellEnvIsTheOldFile(t *testing.T) {
	m, e := machine(t, table(map[string]string{"B": "it's", "A": "plain", "C": "", "D": "a b$x"}))
	if err := write(t, e, units(t, e, "file/shell-env")); err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/env.golden")
	if err != nil {
		t.Fatal(err)
	}
	if got := m.Files[envPath]; string(got) != string(want) {
		t.Errorf("env = %q, want %q", got, want)
	}
	if m.Owners[envPath] != [2]int{0, 1000} || m.Modes[envPath] != 0o640 {
		t.Errorf("owner %v, mode %o; want 0:1000, 640", m.Owners[envPath], m.Modes[envPath])
	}
}

func TestShellEnvWithoutSecretsPlansNothing(t *testing.T) {
	m, e := machine(t, func(context.Context) (any, error) { return nil, kind.ErrNoSecrets })
	m.Files[envPath] = []byte("by hand\n")
	if got := changes(t, units(t, e, "file/shell-env")[0]); got != nil {
		t.Errorf("plan = %q, want none", got)
	}
}

func TestShellEnvRefusesAFileByHand(t *testing.T) {
	m, e := machine(t, table(map[string]string{"A": "new-dummy"}))
	m.Files[envPath] = []byte("export OLD=old-dummy\n")
	_, err := units(t, e, "file/shell-env")[0].Step.Plan(t.Context())
	if err == nil || strings.Contains(err.Error(), "dummy") {
		t.Errorf("plan = %v, want a refusal that shows no value", err)
	}
}

func TestShellEnvGoesWithAnEmptyTable(t *testing.T) {
	m, e := machine(t, table(nil))
	m.Files[envPath] = []byte("# Written by github.com/zadykian/ws\nexport A=dummy\n")
	us := units(t, e, "file/shell-env")
	if err := write(t, e, us); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Files[envPath]; ok {
		t.Error("the file stayed")
	}
	if got := changes(t, us[0]); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
}

func TestBashrcHoldsOneTeraMark(t *testing.T) {
	text, err := os.ReadFile("files/bashrc.tera")
	if err != nil {
		t.Fatal(err)
	}
	s := string(text)
	if strings.Count(s, "{{") != 1 || !strings.Contains(s, "export DOTNET_RUNTIME_ID=linux-{{ arch() }}\n") {
		t.Errorf("bashrc.tera's marks: want {{ arch() }} alone")
	}
	if strings.Contains(s, "{%") || strings.Contains(s, "{#") {
		t.Errorf("bashrc.tera holds a mark of Tera's that bash meant")
	}
}
