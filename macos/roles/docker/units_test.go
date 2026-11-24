package docker_test

import (
	"encoding/json"
	"os"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/macos/facts"
)

const config = "/Users/v/.docker/config.json"

// macUnits loads the role's units on a Mac of the user v.
func macUnits(t *testing.T, m *effecttest.Machine) []kind.Unit {
	fsys := fstest.MapFS{}
	for _, name := range []string{"units.toml.tmpl", "files/config.json"} {
		text, err := os.ReadFile(name) //nolint:gosec // a file of the role, named above
		if err != nil {
			t.Fatal(err)
		}
		fsys["docker/"+name] = &fstest.MapFile{Data: text}
	}
	m.Results["id -u v"] = effecttest.Result{Out: []byte("501\n")}
	m.Results["id -g v"] = effecttest.Result{Out: []byte("20\n")}
	m.Results["dscl . -read /Users/v NFSHomeDirectory"] = effecttest.Result{Out: []byte("NFSHomeDirectory: /Users/v\n")}
	m.Dirs["/Users/v"] = 0o750
	m.Owners["/Users/v"] = [2]int{501, 20}
	units, err := kind.Load(kind.Env{M: m, Out: &effecttest.Out{}, Data: facts.Mac{User: "v", Home: "/Users/v"},
		System: kind.MacOS, User: "v"}, fsys)
	if err != nil {
		t.Fatal(err)
	}
	return units
}

func TestConfigSeedNamesThePluginsAndTheKeychain(t *testing.T) {
	m := effecttest.NewMachine()
	units := macUnits(t, m)
	if _, err := step.Run(t.Context(), kind.Steps(units), &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
	var got struct {
		Dirs  []string `json:"cliPluginsExtraDirs"`
		Store string   `json:"credsStore"`
	}
	if err := json.Unmarshal(m.Files[config], &got); err != nil {
		t.Fatalf("%s: %v", config, err)
	}
	if len(got.Dirs) != 1 || got.Dirs[0] != "/opt/homebrew/lib/docker/cli-plugins" || got.Store != "osxkeychain" {
		t.Errorf("%s = %+v, want the plugins' directory and osxkeychain", config, got)
	}
	if m.Modes[config] != 0o600 || m.Owners[config] != [2]int{501, 20} {
		t.Errorf("%s: mode %o, owner %v; want 0600, v's", config, m.Modes[config], m.Owners[config])
	}
	for _, u := range units {
		if p, err := u.Step.Plan(t.Context()); err != nil || len(p.Changes) > 0 {
			t.Errorf("%s: second plan = %v, %v; want none", u.Name, p.Changes, err)
		}
	}
}

// A seed is the user's after its write: a file there stays, and the plan changes nothing.
func TestConfigSeedLeavesTheUsersFile(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[config], m.Owners[config] = []byte("{}\n"), [2]int{501, 20}
	units := macUnits(t, m)
	for _, u := range units {
		if p, err := u.Step.Plan(t.Context()); err != nil || len(p.Changes) > 0 {
			t.Errorf("%s: plan = %v, %v; want none", u.Name, p.Changes, err)
		}
	}
}
