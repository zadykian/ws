package shell_test

import (
	"os"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/ubuntu/values"
)

// role is the shell's spec, as kind.Load finds a role's.
func role(t *testing.T) fstest.MapFS {
	text, err := os.ReadFile("units.toml.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	return fstest.MapFS{"shell/units.toml.tmpl": &fstest.MapFile{Data: text}}
}

// machine is a machine with root's home and /etc/ws, and its env for the default values.
func machine(t *testing.T) (*effecttest.Machine, kind.Env) {
	v, err := values.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	m := effecttest.NewMachine()
	m.Results["getent passwd root"] = effecttest.Result{Out: []byte("root:x:0:0:root:/root:/bin/bash\n")}
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

// The lines go at the top of root's files.
func TestShellUnitsPutTheLinesAtTheTopOfRootsFiles(t *testing.T) {
	m, e := machine(t)
	m.Files["/root/.bashrc"] = []byte("# stock\n")
	want := []string{"file/shell-bashrc-line", "file/shell-profile-line"}
	if got := kind.Names(units(t, e)); !slices.Equal(got, want) {
		t.Fatalf("units = %q, want %q", got, want)
	}
	us := units(t, e)
	if err := write(t, e, us); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/root/.bashrc"]); got != ". /etc/ws/bashrc\n\n# stock\n" {
		t.Errorf(".bashrc = %q, want the line at the top", got)
	}
	if got := string(m.Files["/root/.profile"]); got != ". /etc/ws/profile\n" {
		t.Errorf(".profile = %q, want the line", got)
	}
	for _, u := range us {
		if got := changes(t, u); got != nil {
			t.Errorf("second plan of %s = %q, want none", u.Name, got)
		}
	}
}
