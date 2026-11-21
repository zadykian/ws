package shell_test

import (
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/values"
)

// role is the shell's spec, as kind.Load finds a role's.
func role(t *testing.T) fstest.MapFS {
	text, err := os.ReadFile("units.toml.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	return fstest.MapFS{"shell/units.toml.tmpl": &fstest.MapFile{Data: text}}
}

// machine is a machine with the account zadykian, 1000:1000, and /etc/ws, and its env for the
// default values.
func machine(t *testing.T) (*effecttest.Machine, kind.Env) {
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
	return m, kind.Env{M: m, U: &effecttest.Units{}, Out: &effecttest.Out{}, Data: v}
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

// The lines go at the top of the account's files, or root's.
func TestShellUnitsForAnAccountAndRoot(t *testing.T) {
	m, e := machine(t)
	m.Files["/home/zadykian/.bashrc"] = []byte("# stock\n")
	want := []string{"file/shell-bashrc-line", "file/shell-profile-line"}
	if got := kind.Names(units(t, e)); !slices.Equal(got, want) {
		t.Fatalf("units = %q, want %q", got, want)
	}
	us := units(t, e)
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
