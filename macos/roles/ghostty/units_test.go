package ghostty_test

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

// config is the config that Ghostty's menu opens on a Mac with none.
const config = "/Users/v/Library/Application Support/com.mitchellh.ghostty/config.ghostty"

// macUnits loads the role's units on a Mac of the user v, with what the machine holds.
func macUnits(t *testing.T, m *effecttest.Machine) []kind.Unit {
	fsys := fstest.MapFS{}
	for _, name := range []string{"units.toml.tmpl", "files/ws.ghostty"} {
		text, err := os.ReadFile(name) //nolint:gosec // a file of the role, named above
		if err != nil {
			t.Fatal(err)
		}
		fsys["ghostty/"+name] = &fstest.MapFile{Data: text}
	}
	m.Results["id -u v"] = effecttest.Result{Out: []byte("501\n")}
	m.Results["id -g v"] = effecttest.Result{Out: []byte("20\n")}
	m.Results["dscl . -read /Users/v NFSHomeDirectory"] = effecttest.Result{Out: []byte("NFSHomeDirectory: /Users/v\n")}
	m.Dirs["/Users/v"] = 0o750
	m.Owners["/Users/v"] = [2]int{501, 20}
	f := facts.Mac{User: "v", Home: "/Users/v", Ghostty: facts.Ghostty{Config: config}}
	units, err := kind.Load(kind.Env{M: m, Out: &effecttest.Out{}, Data: f, System: kind.MacOS, User: "v"}, fsys)
	if err != nil {
		t.Fatal(err)
	}
	return units
}

func TestGhosttyUnitsWriteTheSettingsAndTheLine(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/etc"] = 0o755
	units := macUnits(t, m)
	if _, err := step.Run(t.Context(), kind.Steps(units), &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("files/ws.ghostty")
	if err != nil {
		t.Fatal(err)
	}
	if got := m.Files["/etc/ws/ghostty"]; string(got) != string(want) || !strings.HasPrefix(string(got), "# Written by") {
		t.Errorf("/etc/ws/ghostty = %q, want files/ws.ghostty under ws's mark", got)
	}
	if got := string(m.Files[config]); got != "config-file = ?/etc/ws/ghostty\n" || m.Owners[config] != [2]int{501, 20} {
		t.Errorf("config = %q, owner %v; want the line alone, v's", got, m.Owners[config])
	}
	m.Files[config] = []byte("font-size = 13\nconfig-file = ?/etc/ws/ghostty\n")
	for _, u := range units {
		p, err := u.Step.Plan(t.Context())
		if err != nil || len(p.Changes) > 0 {
			t.Errorf("%s: second plan = %v, %v; want none", u.Name, p.Changes, err)
		}
	}
}

func TestGhosttyUnitsRefuseSettingsByHand(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/etc/ws/ghostty"] = []byte("font-size = 20\n")
	if _, err := step.Run(t.Context(), kind.Steps(macUnits(t, m)), &effecttest.Out{}, false); err == nil ||
		string(m.Files["/etc/ws/ghostty"]) != "font-size = 20\n" {
		t.Errorf("run = %v, file %q; want the file by hand refused and left", err, m.Files["/etc/ws/ghostty"])
	}
}
