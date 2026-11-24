package zed_test

import (
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/macos/facts"
)

const dir = "/Users/v/.config/zed"

// macUnits loads the role's units on a Mac of the user v with the servers named, and returns them
// with the lines they print.
func macUnits(t *testing.T, m *effecttest.Machine, servers ...string) ([]kind.Unit, *effecttest.Out) {
	fsys := fstest.MapFS{}
	for _, name := range []string{"units.toml.tmpl", "files/global_settings.json", "files/keymap.json", "files/settings.json.tmpl"} {
		text, err := os.ReadFile(name) //nolint:gosec // a file of the role, named above
		if err != nil {
			t.Fatal(err)
		}
		fsys["zed/"+name] = &fstest.MapFile{Data: text}
	}
	m.Results["id -u v"] = effecttest.Result{Out: []byte("501\n")}
	m.Results["id -g v"] = effecttest.Result{Out: []byte("20\n")}
	m.Results["dscl . -read /Users/v NFSHomeDirectory"] = effecttest.Result{Out: []byte("NFSHomeDirectory: /Users/v\n")}
	m.Dirs["/Users/v"] = 0o750
	m.Owners["/Users/v"] = [2]int{501, 20}
	f := facts.Mac{User: "v", Home: "/Users/v"}
	for _, s := range servers {
		f.Servers = append(f.Servers, facts.Server{Name: s})
	}
	out := &effecttest.Out{}
	units, err := kind.Load(kind.Env{M: m, Out: out, Data: f, System: kind.MacOS, User: "v"}, fsys)
	if err != nil {
		t.Fatal(err)
	}
	return units, out
}

func run(t *testing.T, units []kind.Unit) {
	if _, err := step.Run(t.Context(), kind.Steps(units), &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
}

// The seed lists each server, as the zed step's seed did.
func TestZedSeedRenders(t *testing.T) {
	m := effecttest.NewMachine()
	units, _ := macUnits(t, m, "lab", "ws")
	run(t, units)
	want := "// Zed's own settings, which its UI writes. ws wrote this once, where none was; ws's\n" +
		"// settings are in global_settings.json, and a key set here wins over them.\n" +
		"{ \"ssh_connections\": [{ \"host\": \"lab\" }, { \"host\": \"ws\" }] }\n"
	if got := string(m.Files[dir+"/settings.json"]); got != want {
		t.Errorf("settings.json = %q, want %q", got, want)
	}
	for _, name := range []string{"global_settings.json", "keymap.json"} {
		text, err := os.ReadFile("files/" + name) //nolint:gosec // a file of the role, named above
		if err != nil {
			t.Fatal(err)
		}
		if got := m.Files[dir+"/"+name]; string(got) != string(text) || m.Owners[dir+"/"+name] != [2]int{501, 20} {
			t.Errorf("%s = %q, owner %v; want files/%s, v's", name, got, m.Owners[dir+"/"+name], name)
		}
	}
	m.Files[dir+"/settings.json"] = []byte("{ \"theme\": \"mine\" }\n")
	for _, u := range units {
		if p, err := u.Step.Plan(t.Context()); err != nil || len(p.Changes) > 0 {
			t.Errorf("%s: second plan = %v, %v; want none", u.Name, p.Changes, err)
		}
	}
}

// A keymap.json of the user's own is left with one warning, and ws's comes back once it goes.
func TestZedKeepsTheUsersKeymap(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[dir+"/keymap.json"] = []byte("[]\n")
	units, out := macUnits(t, m)
	run(t, units)
	if got := string(m.Files[dir+"/keymap.json"]); got != "[]\n" {
		t.Errorf("keymap.json = %q, want the user's left", got)
	}
	warned := slices.IndexFunc(out.Lines, func(l string) bool { return strings.Contains(l, "keymap.json lacks the line") })
	if warned < 0 {
		t.Errorf("lines = %q, want a warning of the user's keymap.json", out.Lines)
	}
	delete(m.Files, dir+"/keymap.json")
	run(t, units)
	if !strings.HasPrefix(string(m.Files[dir+"/keymap.json"]), "// Written by github.com/zadykian/ws") {
		t.Errorf("keymap.json = %q, want ws's written back", m.Files[dir+"/keymap.json"])
	}
}
