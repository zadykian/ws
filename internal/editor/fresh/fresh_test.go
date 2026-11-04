package fresh

import (
	"bytes"
	"encoding/json"
	"os"
	"slices"
	"strings"
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

// typed parses config.json into the struct, to read the bindings.
func typed(t *testing.T) config {
	data, err := Config(stubPaths)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := bytes.CutPrefix(data, []byte(header))
	var c config
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

func TestEveryTargetIsAnActionOfFresh(t *testing.T) {
	data, err := os.ReadFile("testdata/actions.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, list, _ := strings.Cut(string(data), "\n")
	known := strings.Fields(list)
	targets, err := editor.LoadTargets(actionFiles, "actions/*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for action, target := range targets {
		if target.To != "" && !slices.Contains(known, target.To) {
			t.Errorf("%s: %s isn't an action of Fresh", action, target.To)
		}
	}
}

func TestEveryActionHasATargetOrAReason(t *testing.T) {
	keymap, err := editor.LoadKeymap()
	if err != nil {
		t.Fatal(err)
	}
	targets, err := editor.LoadTargets(actionFiles, "actions/*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := keymap.Check(targets); err != nil {
		t.Fatal(err)
	}
}

func TestAStrokeIsSpeltAsFreshSpellsIt(t *testing.T) {
	for _, c := range []struct {
		stroke string
		want   press
	}{
		{"ctrl+s", press{"s", []string{"ctrl"}}},
		{"f5", press{"F5", nil}},
		{"ctrl+alt+shift+pageup", press{"PageUp", []string{"ctrl", "alt", "shift"}}},
		{"escape", press{"Esc", nil}},
		{"cmd+space", press{"Space", []string{"super"}}},
		{"ctrl+[", press{"[", []string{"ctrl"}}},
	} {
		chord, err := editor.ParseChord(c.stroke)
		if err != nil {
			t.Fatal(err)
		}
		got := forms(chord[0])[len(forms(chord[0]))-1]
		if got.Key != c.want.Key || !slices.Equal(got.Modifiers, c.want.Modifiers) {
			t.Errorf("%s = %+v; want %+v", c.stroke, got, c.want)
		}
	}
}

func TestAShiftedSignTakesTwoForms(t *testing.T) {
	chord, err := editor.ParseChord("ctrl+shift+/")
	if err != nil {
		t.Fatal(err)
	}
	got := forms(chord[0])
	if len(got) != 2 || got[0].Key != "?" || slices.Contains(got[0].Modifiers, "shift") {
		t.Errorf("forms = %+v; want the sign with no Shift, then the key with it", got)
	}
}

func TestAnArrivalTakesNoBoundKey(t *testing.T) {
	keymap, err := editor.LoadKeymap()
	if err != nil {
		t.Fatal(err)
	}
	for _, scope := range editor.Scopes {
		for _, b := range keymap[scope] {
			for _, chord := range b.Keys {
				for from, to := range arrivals {
					if chord.String() == to && !slices.Contains(skipped, to) {
						t.Errorf("%s arrives as %s, which %s binds", from, to, b.Action)
					}
				}
			}
		}
	}
}

func TestTheActiveMapResolves(t *testing.T) {
	c := typed(t)
	m, found := c.Maps[c.ActiveMap]
	if !found || m.Inherits != "default" || len(m.Bindings) == 0 {
		t.Fatalf("map %q: %+v", c.ActiveMap, m)
	}
}

func TestNoKeyFiresTwoActions(t *testing.T) {
	seen := map[string]string{}
	for _, b := range typed(t).Maps["f12"].Bindings {
		form, err := json.Marshal([]any{b.When, b.Key, b.Modifiers, b.Keys})
		if err != nil {
			t.Fatal(err)
		}
		if other, taken := seen[string(form)]; taken {
			t.Errorf("%s binds both %s and %s", form, other, b.Action)
		}
		seen[string(form)] = b.Action
	}
}
