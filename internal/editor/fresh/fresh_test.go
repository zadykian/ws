package fresh

import (
	"bytes"
	"encoding/json"
	"slices"
	"testing"

	"github.com/zadykian/ws/internal/editor"
)

var stubPaths = map[string]string{
	"gopls": "/g", "roslyn-language-server": "/r", "postgres-language-server": "/p",
	"ansible-language-server": "/a", "vscode-html-language-server": "/h",
	"dockerfile-language-server": "/d", "docker-language-server": "/k",
}

// rendered parses config.json into a map, as a struct's zero value hides a missing key.
func rendered(t *testing.T) map[string]any {
	data, err := Config(stubPaths)
	if err != nil {
		t.Fatal(err)
	}
	body, found := bytes.CutPrefix(data, []byte(header))
	if !found {
		t.Fatal("no header")
	}
	var c map[string]any
	if err := json.Unmarshal(body, &c); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestConfigTurnsUpdatesAndSessionsOff(t *testing.T) {
	c := rendered(t)
	for _, key := range []string{"check_for_updates", "self_update", "auto_attach"} {
		if got, found := c[key]; !found || got != false {
			t.Errorf("%s = %v, found %v; want false", key, got, found)
		}
	}
}

func TestTheSharedSettingsReachTheEditor(t *testing.T) {
	settings, err := editor.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	table, _ := rendered(t)["editor"].(map[string]any)
	if rulers, _ := table["rulers"].([]any); len(rulers) != 1 || rulers[0] != float64(settings.Ruler) {
		t.Errorf("rulers = %v; want [%d]", table["rulers"], settings.Ruler)
	}
	for _, key := range []string{
		"whitespace_show", "whitespace_spaces_leading", "whitespace_spaces_inner",
		"whitespace_spaces_trailing", "whitespace_tabs_inner", "whitespace_tabs_trailing",
	} {
		if table[key] != settings.VisibleWhitespace {
			t.Errorf("%s = %v; want %v", key, table[key], settings.VisibleWhitespace)
		}
	}
	if table["auto_save_enabled"] != (settings.AutoSaveSeconds > 0) ||
		table["auto_save_interval_secs"] != float64(settings.AutoSaveSeconds) {
		t.Errorf("auto-save = %v, %v; want every %d seconds", table["auto_save_enabled"],
			table["auto_save_interval_secs"], settings.AutoSaveSeconds)
	}
	if table["rainbow_brackets"] != false {
		t.Errorf("rainbow_brackets = %v", table["rainbow_brackets"])
	}
}

func TestAServerWithNoPathFails(t *testing.T) {
	paths := map[string]string{}
	for name, path := range stubPaths {
		paths[name] = path
	}
	delete(paths, "gopls")
	if _, err := Config(paths); err == nil {
		t.Error("no error")
	}
}

func TestAComposeFileTakesItsLanguageID(t *testing.T) {
	c := rendered(t)
	langs, _ := c["languages"].(map[string]any)
	compose, _ := langs["docker-compose"].(map[string]any)
	names, _ := compose["filenames"].([]any)
	for _, file := range composeFiles {
		if !slices.Contains(names, any(file)) {
			t.Errorf("docker-compose lacks the file %s", file)
		}
	}
	servers, _ := c["lsp"].(map[string]any)
	for language, list := range servers {
		for _, entry := range list.([]any) {
			ids := entry.(map[string]any)["language_id_overrides"]
			if (ids != nil) != (language == "docker-compose") {
				t.Errorf("%s: language_id_overrides = %v", language, ids)
			}
		}
	}
}

func TestEveryLanguageHasAName(t *testing.T) {
	catalogue, err := editor.LoadServers()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range catalogue {
		for _, language := range s.Languages {
			if _, found := languages[language]; !found {
				t.Errorf("%s: %s has no name in Fresh", s.Name, language)
			}
		}
	}
}
