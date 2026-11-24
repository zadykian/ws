package devtools_test

import (
	"os"
	"testing"
	"testing/fstest"

	"github.com/pelletier/go-toml/v2"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/macos/facts"
)

// macUnits loads the role's units on a Mac of the user v.
func macUnits(t *testing.T, m *effecttest.Machine) []kind.Unit {
	fsys := fstest.MapFS{}
	for _, name := range []string{"units.toml.tmpl"} {
		text, err := os.ReadFile(name) //nolint:gosec // a file of the role, named above
		if err != nil {
			t.Fatal(err)
		}
		fsys["devtools/"+name] = &fstest.MapFile{Data: text}
	}
	m.Results["id -u v"] = effecttest.Result{Out: []byte("501\n")}
	m.Results["id -g v"] = effecttest.Result{Out: []byte("20\n")}
	m.Results["dscl . -read /Users/v NFSHomeDirectory"] = effecttest.Result{Out: []byte("NFSHomeDirectory: /Users/v\n")}
	m.Dirs["/Users/v"], m.Dirs["/usr/local"] = 0o750, 0o755
	m.Owners["/Users/v"] = [2]int{501, 20}
	units, err := kind.Load(kind.Env{M: m, Out: &effecttest.Out{}, Data: facts.Mac{User: "v", Home: "/Users/v"},
		System: kind.MacOS, User: "v"}, fsys)
	if err != nil {
		t.Fatal(err)
	}
	return units
}

func TestGoIsLinkedToMisesInstall(t *testing.T) {
	m := effecttest.NewMachine()
	units := macUnits(t, m)
	if _, err := step.Run(t.Context(), kind.Steps(units), &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
	text, err := os.ReadFile("mise.macos.toml")
	if err != nil {
		t.Fatal(err)
	}
	var c struct {
		Tools map[string]string `toml:"tools"`
	}
	if err := toml.Unmarshal(text, &c); err != nil {
		t.Fatal(err)
	}
	want := "/Users/v/.local/share/mise/installs/go/" + c.Tools["go"]
	if got := m.Links["/usr/local/go"]; got != want {
		t.Errorf("/usr/local/go -> %q, want %q", got, want)
	}
	for _, u := range units {
		if p, err := u.Step.Plan(t.Context()); err != nil || len(p.Changes) > 0 {
			t.Errorf("%s: second plan = %v, %v; want none", u.Name, p.Changes, err)
		}
	}
}
