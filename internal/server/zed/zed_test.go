package zed

import (
	"context"
	"encoding/json"
	"io/fs"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/devtools"
	"github.com/zadykian/ws/internal/server/values"
)

// valuesOf returns the defaults for the user and the home.
func valuesOf(user, home string) values.Values {
	v := values.Default()
	v.User, v.Home = user, home
	return v
}

// run applies the role on a machine with an empty home for the user.
func run(t *testing.T, user, home string) *effecttest.Machine {
	t.Helper()
	m := effecttest.NewMachine()
	m.Dirs[home] = 0o755
	m.Results["id -u "+user] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["id -g "+user] = effecttest.Result{Out: []byte("1000\n")}
	steps := Steps(valuesOf(user, home), secrets.Secrets{}, m, &effecttest.Units{}, &effecttest.Out{})
	if _, err := step.Run(context.Background(), steps, &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestSettingsNameTheToolPathsUnderTheHome(t *testing.T) {
	for _, home := range []string{"/root", "/home/ws"} {
		v := values.Default()
		v.Home = home
		got, err := settingsFile(v)
		if err != nil {
			t.Fatal(err)
		}
		_, body, _ := strings.Cut(string(got), "\n")
		if !json.Valid([]byte(body)) {
			t.Fatalf("not JSON:\n%s", body)
		}
		for _, want := range []string{devtools.Gopls(home), devtools.Roslyn(home), devtools.Dlv(home)} {
			if !strings.Contains(body, `"`+want+`"`) {
				t.Errorf("%s: settings lack %q", home, want)
			}
		}
	}
}

func TestSettingsAreOwnedByTheUserOnAnEmptyHome(t *testing.T) {
	m := run(t, "ws", "/home/ws")
	for _, p := range []string{"/home/ws/.config/zed/settings.json"} {
		if m.Owners[p] != [2]int{1000, 1000} {
			t.Errorf("%s owner %v", p, m.Owners[p])
		}
	}
	if m.Dirs["/home/ws/.config"] != 0o755 || m.Dirs["/home/ws/.config/zed"] != 0o755 {
		t.Errorf("dirs %v", m.Dirs)
	}
}

func TestASecondRunChangesNothing(t *testing.T) {
	m := run(t, "root", "/root")
	steps := Steps(valuesOf("root", "/root"), secrets.Secrets{}, m, &effecttest.Units{}, &effecttest.Out{})
	changed, err := step.Run(context.Background(), steps, &effecttest.Out{}, true)
	if err != nil || changed {
		t.Errorf("second run: %v, %v", changed, err)
	}
}

func TestTheLimitNamesTheUserAndTheValue(t *testing.T) {
	m := run(t, "ws", "/home/ws")
	_, line, _ := strings.Cut(string(m.Files[limitsPath]), "\n")
	if line != "ws soft nofile 65536\n" {
		t.Errorf("limits line %q", line)
	}
	if m.Owners[limitsPath] != [2]int{} {
		t.Errorf("owner %v", m.Owners[limitsPath])
	}
}

func TestSettingsNameTheSQLAndDockerServersWithNoDuplicateKey(t *testing.T) {
	got, err := settingsFile(values.Default())
	if err != nil {
		t.Fatal(err)
	}
	_, body, _ := strings.Cut(string(got), "\n")
	var top map[string]map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &top); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"postgres-language-server", "docker-language-server"} {
		if _, ok := top["lsp"][key]; !ok {
			t.Errorf("lsp lacks %s", key)
		}
	}
	for _, key := range []string{"lsp", "languages", "file_types", "dap"} {
		if strings.Count(body, "\n  \""+key+"\":") != 1 {
			t.Errorf("top-level key %s is not unique", key)
		}
	}
}

func TestSettingsNameTheNpmServersForRootAndAnotherUser(t *testing.T) {
	for home, bin := range map[string]string{"/root": "/root", "/home/ws": "/home/ws"} {
		v := values.Default()
		v.Home = home
		got, err := settingsFile(v)
		if err != nil {
			t.Fatal(err)
		}
		_, body, _ := strings.Cut(string(got), "\n")
		if !json.Valid([]byte(body)) {
			t.Fatalf("not JSON:\n%s", body)
		}
		for _, name := range []string{"ansible-language-server", "vscode-html-language-server", "docker-langserver"} {
			want := `"` + bin + "/.local/lib/language-servers/node_modules/.bin/" + name + `"`
			if !strings.Contains(body, want) {
				t.Errorf("%s: settings lack %s", home, want)
			}
		}
		for _, key := range []string{"lsp", "languages", "node"} {
			if strings.Count(body, "\n  \""+key+"\":") != 1 {
				t.Errorf("top-level key %s is not unique", key)
			}
		}
	}
}

func TestTheStepsRefuseALinkAtAFileRootWrites(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/home/ws"] = 0o755
	m.Results["id -u ws"] = effecttest.Result{Out: []byte("1000\n")}
	m.Results["id -g ws"] = effecttest.Result{Out: []byte("1000\n")}
	m.Links["/home/ws/.config/zed"] = "/srv/target"
	steps := Steps(valuesOf("ws", "/home/ws"), secrets.Secrets{}, m, &effecttest.Units{}, &effecttest.Out{})
	if _, err := step.Run(context.Background(), steps, &effecttest.Out{}, false); err == nil {
		t.Error("a link at the settings' directory: no error")
	}
}

func TestTheTerminalFilesAreRootsAtTheirModes(t *testing.T) {
	m := run(t, "ws", "/home/ws")
	for path, want := range map[string]fs.FileMode{
		"/usr/local/lib/ws/zed/terminal": 0o755, "/usr/local/lib/ws/zed/tmux.conf": 0o644,
	} {
		if len(m.Files[path]) == 0 || m.Owners[path] != [2]int{} {
			t.Errorf("%s: %d bytes, owner %v", path, len(m.Files[path]), m.Owners[path])
		}
		if got, set := m.Modes[path]; want != 0o644 && (!set || got != want) {
			t.Errorf("%s: mode %v", path, got)
		}
	}
	for _, dir := range []string{"/usr/local/lib/ws", "/usr/local/lib/ws/zed"} {
		if m.Dirs[dir] != 0o755 || m.Owners[dir] != [2]int{} {
			t.Errorf("%s: mode %v, owner %v", dir, m.Dirs[dir], m.Owners[dir])
		}
	}
}

func TestTheTerminalStepComesFirstAndMakesItsParents(t *testing.T) {
	m := effecttest.NewMachine()
	steps := Steps(valuesOf("ws", "/home/ws"), secrets.Secrets{}, m, &effecttest.Units{}, &effecttest.Out{})
	if steps[0].Name() != "Zed terminal" {
		t.Fatalf("the first step is %q", steps[0].Name())
	}
	plan, err := steps[0].Plan(context.Background())
	if err != nil || len(plan.Changes) != 4 {
		t.Errorf("an empty machine plans %d changes, %v", len(plan.Changes), err)
	}
}

func TestSettingsRunTheWrapperAsTheTerminalProgramAfterItsStep(t *testing.T) {
	got, err := settingsFile(values.Default())
	if err != nil {
		t.Fatal(err)
	}
	_, body, _ := strings.Cut(string(got), "\n")
	var top struct {
		Terminal struct{ Shell struct{ Program string } }
	}
	if err := json.Unmarshal([]byte(body), &top); err != nil {
		t.Fatal(err)
	}
	if top.Terminal.Shell.Program != terminalPath {
		t.Errorf("program %q, want %q", top.Terminal.Shell.Program, terminalPath)
	}
	steps := Steps(valuesOf("ws", "/home/ws"), secrets.Secrets{}, effecttest.NewMachine(), &effecttest.Units{}, &effecttest.Out{})
	if steps[1].Name() != "Zed settings" || steps[1].WaitsOn() != steps[0].Name() {
		t.Errorf("%q waits on %q", steps[1].Name(), steps[1].WaitsOn())
	}
}
