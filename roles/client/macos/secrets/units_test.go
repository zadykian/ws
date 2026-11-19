package secrets_test

import (
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/roles/client/facts"
)

// Where mise installs the tree's fnox and age-plugin-yubikey.
const (
	misesFnox   = "/Users/v/.local/share/mise/installs/github-jdx-fnox/1.50.0/fnox"
	misesPlugin = "/Users/v/.local/share/mise/installs/age-plugin-yubikey/0.5.1/age-plugin-yubikey"
)

// runs is the prefix of each line that a state unit runs, with the PATH it gives.
const runs = "env PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin sh -c "

// copyLines returns the lines of the tool name's copy in /usr/local/bin, each with mise's path
// from. The check runs as the user, and the copy as root.
func copyLines(name, from string) (check, cp string) {
	dst := "/usr/local/bin/" + name
	tmp := "/usr/local/bin/." + name + ".new"
	check = runs + "cmp -s \"$1\" " + dst + " && [ \"$(stat -f %u:%g:%Lp " + dst + ")\" = 0:0:755 ] sh " + from
	cp = runs + "test -x \"$1\" && mkdir -p /usr/local/bin && install -o root -g wheel -m 0755 \"$1\" " + tmp +
		" && mv -f " + tmp + " " + dst + " sh " + from
	return check, cp
}

// macUnits loads the role's units on a Mac of the user v, with mise's tools.
func macUnits(t *testing.T, m *effecttest.Machine) []kind.Unit {
	text, err := os.ReadFile("units.toml.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	fsys := fstest.MapFS{"secrets/units.toml.tmpl": &fstest.MapFile{Data: text}}
	units, err := kind.Load(kind.Env{M: m, Out: &effecttest.Out{}, System: kind.MacOS, User: "v",
		Data: facts.Mac{User: "v", Home: "/Users/v",
			Secrets: facts.Secrets{FnoxFrom: misesFnox, PluginFrom: misesPlugin}}}, fsys)
	if err != nil {
		t.Fatal(err)
	}
	return units
}

// exitOne is the error of a command that exits 1.
func exitOne() error {
	return exec.Command("sh", "-c", "exit 1").Run()
}

func TestSecretsUnitsCopyEachTool(t *testing.T) {
	m := effecttest.NewMachine()
	m.After = map[string]map[string]effecttest.Result{}
	tools := map[string]string{"fnox": misesFnox, "age-plugin-yubikey": misesPlugin}
	for name, from := range tools {
		check, cp := copyLines(name, from)
		m.Results[check] = effecttest.Result{Err: exitOne()}
		m.Results[cp] = effecttest.Result{}
		m.After[cp] = map[string]effecttest.Result{check: {}}
	}
	units := macUnits(t, m)
	for _, u := range units {
		p, err := u.Step.Plan(t.Context())
		if err == nil {
			err = u.Step.Apply(t.Context(), p)
		}
		if err != nil {
			t.Fatalf("%s: %v", u.Name, err)
		}
	}
	for name, from := range tools {
		check, cp := copyLines(name, from)
		if !slices.Contains(m.Calls, "run "+check) || !slices.Contains(m.Calls, "root "+cp) {
			t.Errorf("calls = %q, want %s's check as the user and its copy through sudo", m.Calls, name)
		}
	}
	m.Calls = nil
	for _, u := range units {
		if p, err := u.Step.Plan(t.Context()); err != nil || len(p.Changes) > 0 {
			t.Errorf("%s: second plan = %v, %v; want none", u.Name, p.Changes, err)
		}
	}
	for _, call := range m.Calls {
		if strings.HasPrefix(call, "root ") {
			t.Errorf("a plan with each copy in place ran %q, want no copy", call)
		}
	}
}
