package zed_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// terminalPath is the wrapper, which the settings name as the terminal's program.
const terminalPath = "/usr/local/lib/ws/zed/terminal"

// body returns the settings below their comment line, which must be JSON.
func body(t *testing.T) string {
	data, err := os.ReadFile("files/settings.json")
	if err != nil {
		t.Fatal(err)
	}
	comment, rest, _ := strings.Cut(string(data), "\n")
	if !strings.HasPrefix(comment, "//") {
		t.Fatalf("no comment line first: %q", comment)
	}
	if !json.Valid([]byte(rest)) {
		t.Fatalf("not JSON:\n%s", rest)
	}
	return rest
}

func TestSettingsAreJSONUnderTheirCommentLine(t *testing.T) {
	body(t)
}

// The settings run the wrapper that mise writes, at the mode that lets Zed run it.
func TestSettingsRunTheWrapperThatMiseWrites(t *testing.T) {
	var top struct {
		Terminal struct{ Shell struct{ Program string } }
	}
	if err := json.Unmarshal([]byte(body(t)), &top); err != nil {
		t.Fatal(err)
	}
	if top.Terminal.Shell.Program != terminalPath {
		t.Errorf("program %q, want %q", top.Terminal.Shell.Program, terminalPath)
	}
	text, err := os.ReadFile("mise.guest.toml")
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Bootstrap struct {
			Files map[string]struct{ Source, Mode string }
		}
	}
	if err := toml.Unmarshal(text, &config); err != nil {
		t.Fatal(err)
	}
	if f := config.Bootstrap.Files[terminalPath]; f.Source != "files/terminal" || f.Mode != "0755" {
		t.Errorf("mise's entry for %s = %+v", terminalPath, f)
	}
}

func TestSettingsNameTheSQLAndDockerServersWithNoDuplicateKey(t *testing.T) {
	text := body(t)
	var top map[string]map[string]json.RawMessage
	if err := json.Unmarshal([]byte(text), &top); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"postgres-language-server", "docker-language-server"} {
		if _, ok := top["lsp"][key]; !ok {
			t.Errorf("lsp lacks %s", key)
		}
	}
	for _, key := range []string{"lsp", "languages", "file_types", "dap"} {
		if strings.Count(text, "\n  \""+key+"\":") != 1 {
			t.Errorf("top-level key %s is not unique", key)
		}
	}
}

func TestSettingsNameTheNpmServersUnderRootsHome(t *testing.T) {
	text := body(t)
	for _, name := range []string{"ansible-language-server", "vscode-html-language-server", "docker-langserver"} {
		want := `"/root/.local/lib/language-servers/node_modules/.bin/` + name + `"`
		if !strings.Contains(text, want) {
			t.Errorf("settings lack %s", want)
		}
	}
	for _, key := range []string{"lsp", "languages", "node"} {
		if strings.Count(text, "\n  \""+key+"\":") != 1 {
			t.Errorf("top-level key %s is not unique", key)
		}
	}
}
