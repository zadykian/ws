package ikeward_test

import (
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/macos/facts"
)

// misesIkeward is where mise installs the tree's ikeward.
const misesIkeward = "/Users/v/.local/share/mise/installs/github-zadykian-ikeward/0.4.0/ikeward"

// runs is the prefix of each line that a state unit runs, with the PATH it gives.
const runs = "env PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin sh -c "

// The copy's check, which runs as the user, and its command, which runs as root, with mise's path.
const (
	dst   = "/usr/local/sbin/ikeward"
	tmp   = "/usr/local/sbin/.ikeward.new"
	check = runs + "cmp -s \"$1\" " + dst + " && [ \"$(stat -f %u:%g:%Lp /usr/local/sbin " + dst +
		" | tr '\\n' ' ')\" = '0:0:755 0:0:755 ' ] sh " + misesIkeward
	cp = runs + "test -x \"$1\" && mkdir -p /usr/local/sbin && chown root:wheel /usr/local/sbin" +
		" && chmod 0755 /usr/local/sbin && install -o root -g wheel -m 0755 \"$1\" " + tmp +
		" && mv -f " + tmp + " " + dst + " sh " + misesIkeward
)

// macUnits loads the role's units on a Mac of the user v, with mise's ikeward.
func macUnits(t *testing.T, m *effecttest.Machine) []kind.Unit {
	text, err := os.ReadFile("units.toml.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	fsys := fstest.MapFS{"ikeward/units.toml.tmpl": &fstest.MapFile{Data: text}}
	units, err := kind.Load(kind.Env{M: m, Out: &effecttest.Out{}, System: kind.MacOS, User: "v",
		Data: facts.Mac{User: "v", Home: "/Users/v", Ikeward: facts.Ikeward{From: misesIkeward}}}, fsys)
	if err != nil {
		t.Fatal(err)
	}
	return units
}

// exitOne is the error of a command that exits 1.
func exitOne() error {
	return exec.Command("sh", "-c", "exit 1").Run()
}

func TestIkewardUnitCopiesTheBinaryToRoots(t *testing.T) {
	m := effecttest.NewMachine()
	m.After = map[string]map[string]effecttest.Result{}
	m.Results[check] = effecttest.Result{Err: exitOne()}
	m.Results[cp] = effecttest.Result{}
	m.After[cp] = map[string]effecttest.Result{check: {}}
	units := macUnits(t, m)
	if names := kind.Names(units); !slices.Equal(names, []string{"state/ikeward"}) {
		t.Fatalf("units = %v, want state/ikeward alone", names)
	}
	for _, u := range units {
		p, err := u.Step.Plan(t.Context())
		if err == nil {
			err = u.Step.Apply(t.Context(), p)
		}
		if err != nil {
			t.Fatalf("%s: %v", u.Name, err)
		}
	}
	if !slices.Contains(m.Calls, "run "+check) || !slices.Contains(m.Calls, "root "+cp) {
		t.Errorf("calls = %q, want the check as the user and the copy through sudo", m.Calls)
	}
	m.Calls = nil
	for _, u := range units {
		if p, err := u.Step.Plan(t.Context()); err != nil || len(p.Changes) > 0 {
			t.Errorf("%s: second plan = %v, %v; want none", u.Name, p.Changes, err)
		}
	}
	for _, call := range m.Calls {
		if strings.HasPrefix(call, "root ") {
			t.Errorf("a plan with the copy in place ran %q, want no copy", call)
		}
	}
}
