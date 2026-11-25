package zsh_test

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/macos/facts"
)

// macUnits loads the role's units on a Mac of the user v.
func macUnits(t *testing.T, m *effecttest.Machine) []kind.Unit {
	fsys := fstest.MapFS{}
	for _, name := range []string{"units.toml.tmpl", "files/zshrc"} {
		text, err := os.ReadFile(name) //nolint:gosec // a file of the role, named above
		if err != nil {
			t.Fatal(err)
		}
		fsys["zsh/"+name] = &fstest.MapFile{Data: text}
	}
	m.Results["id -u v"] = effecttest.Result{Out: []byte("501\n")}
	m.Results["id -g v"] = effecttest.Result{Out: []byte("20\n")}
	m.Results["dscl . -read /Users/v NFSHomeDirectory"] = effecttest.Result{Out: []byte("NFSHomeDirectory: /Users/v\n")}
	m.Dirs["/Users/v"], m.Dirs["/etc"] = 0o750, 0o755
	m.Owners["/Users/v"] = [2]int{501, 20}
	units, err := kind.Load(kind.Env{M: m, Out: &effecttest.Out{}, Data: facts.Mac{User: "v", Home: "/Users/v"},
		System: kind.MacOS, User: "v"}, fsys)
	if err != nil {
		t.Fatal(err)
	}
	return units
}

func TestZshUnitsWriteTheHookAndEndZshrc(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/Users/v/.zshrc"], m.Owners["/Users/v/.zshrc"] = []byte("export A=1"), [2]int{501, 20}
	units := macUnits(t, m)
	if _, err := step.Run(t.Context(), kind.Steps(units), &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
	hook, err := os.ReadFile("files/zshrc")
	if err != nil {
		t.Fatal(err)
	}
	if got := m.Files["/etc/ws/zshrc"]; string(got) != string(hook) {
		t.Errorf("/etc/ws/zshrc = %q, want files/zshrc", got)
	}
	if got := string(m.Files["/Users/v/.zshrc"]); got != "export A=1\nsource /etc/ws/zshrc\n" {
		t.Errorf("~/.zshrc = %q, want the line last", got)
	}
	for _, u := range units {
		if p, err := u.Step.Plan(t.Context()); err != nil || len(p.Changes) > 0 {
			t.Errorf("%s: second plan = %v, %v; want none", u.Name, p.Changes, err)
		}
	}
}

func TestZshLineRefusesALinkedZshrcThatLacksIt(t *testing.T) {
	m := effecttest.NewMachine()
	m.Links["/Users/v/.zshrc"] = "/Users/v/dots/zshrc"
	// The fake's file at the path stands for what the read through the link gives.
	m.Files["/Users/v/.zshrc"], m.Files["/Users/v/dots/zshrc"] = []byte("export A=1\n"), []byte("export A=1\n")
	_, err := step.Run(t.Context(), kind.Steps(macUnits(t, m)), &effecttest.Out{}, false)
	if err == nil || !strings.Contains(err.Error(), "is a link to /Users/v/dots/zshrc") {
		t.Errorf("run = %v, want the link refused", err)
	}
	if got := string(m.Files["/Users/v/dots/zshrc"]); got != "export A=1\n" {
		t.Errorf("the link's file = %q, want it left", got)
	}
}
