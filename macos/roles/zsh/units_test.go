package zsh_test

import (
	"os"
	"regexp"
	"strings"
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

func TestZshUnitsWriteTheHookThatZshrcSources(t *testing.T) {
	m := effecttest.NewMachine()
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
	for _, u := range units {
		if p, err := u.Step.Plan(t.Context()); err != nil || len(p.Changes) > 0 {
			t.Errorf("%s: second plan = %v, %v; want none", u.Name, p.Changes, err)
		}
	}
	spec, err := os.ReadFile("mise.macos.toml")
	if err != nil {
		t.Fatal(err)
	}
	var c struct {
		Dotfiles map[string]map[string]any `toml:"dotfiles"`
	}
	if err := toml.Unmarshal(spec, &c); err != nil {
		t.Fatal(err)
	}
	// The line that mise adds to ~/.zshrc sources the file that the unit writes.
	if got := c.Dotfiles["~/.zshrc/ws"]["line"]; got != "source /etc/ws/zshrc" {
		t.Errorf("mise.macos.toml's line of ~/.zshrc = %q, want the one that sources /etc/ws/zshrc", got)
	}
}

// pluginsLine is the line of files/zshrc that names oh-my-zsh's plugins.
var pluginsLine = regexp.MustCompile(`(?m)^\s*plugins=\((.*)\)$`)

func TestZshrcLoadsThePluginsThatMiseClones(t *testing.T) {
	rc, err := os.ReadFile("files/zshrc")
	if err != nil {
		t.Fatal(err)
	}
	m := pluginsLine.FindSubmatch(rc)
	if m == nil {
		t.Fatal("files/zshrc has no plugins line")
	}
	plugins := strings.Fields(string(m[1]))
	if got := plugins[len(plugins)-1]; got != "zsh-syntax-highlighting" {
		t.Errorf("plugins = %q, want zsh-syntax-highlighting last", plugins)
	}
	if !strings.Contains(string(rc), "zstyle ':omz:update' mode disabled") {
		t.Error("files/zshrc leaves oh-my-zsh's update on, which would move the checkout off its pin")
	}
	spec, err := os.ReadFile("mise.macos.toml")
	if err != nil {
		t.Fatal(err)
	}
	// mise clones in the file's order, and refuses a directory with files in it.
	omz := strings.Index(string(spec), `"~/.oh-my-zsh" =`)
	for _, name := range plugins {
		if name == "git" {
			continue // oh-my-zsh's own
		}
		at := strings.Index(string(spec), `"~/.oh-my-zsh/custom/plugins/`+name+`" =`)
		if at < 0 || omz < 0 || at < omz {
			t.Errorf("mise.macos.toml does not clone %s after oh-my-zsh", name)
		}
	}
}
