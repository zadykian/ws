package fresh

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/resource"
)

// A malformed config makes Fresh load its defaults and exit 0, so these tests are its only check.

// press is a stroke as Fresh's config spells it.
type press struct {
	Key       string   `json:"key"`
	Modifiers []string `json:"modifiers"`
}

type binding struct {
	press
	Keys   []press `json:"keys"`
	Action string  `json:"action"`
	When   string  `json:"when"`
}

type keymap struct {
	Inherits string    `json:"inherits"`
	Bindings []binding `json:"bindings"`
}

type config struct {
	Active      string            `json:"active_keybinding_map"`
	Maps        map[string]keymap `json:"keybinding_maps"`
	Keybindings []binding         `json:"keybindings"`
}

// withoutComments drops the lines whose first non-blank characters are //, which Fresh skips.
func withoutComments(data []byte) []byte {
	var out bytes.Buffer
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		if !strings.HasPrefix(strings.TrimSpace(sc.Text()), "//") {
			out.Write(sc.Bytes())
			out.WriteByte('\n')
		}
	}
	return out.Bytes()
}

// renderedAt renders the template for the home.
func renderedAt(t *testing.T, home string) []byte {
	data, err := resource.Render(templates, "files/config.json.tmpl", struct{ Home string }{home})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// load parses the render into a map, as a struct's zero value hides a missing key, and into the
// struct, to read the bindings.
func load(t *testing.T) (map[string]any, config) {
	body := withoutComments(renderedAt(t, "/home/ws"))
	var loose map[string]any
	if err := json.Unmarshal(body, &loose); err != nil {
		t.Fatal(err)
	}
	var c config
	if err := json.Unmarshal(body, &c); err != nil {
		t.Fatal(err)
	}
	return loose, c
}

// spell gives a stroke as modifiers and key joined by +, in the order the file lists them.
func spell(p press) string {
	return strings.Join(append(slices.Clone(p.Modifiers), p.Key), "+")
}

// spellAll gives a binding's strokes, joined by a space for a chord.
func (b binding) spellAll() string {
	if len(b.Keys) == 0 {
		return spell(b.press)
	}
	var parts []string
	for _, p := range b.Keys {
		parts = append(parts, spell(p))
	}
	return strings.Join(parts, " ")
}

func TestTheConfigTurnsUpdatesAndSessionsOff(t *testing.T) {
	loose, _ := load(t)
	for _, key := range []string{"check_for_updates", "self_update", "auto_attach"} {
		if got, found := loose[key]; !found || got != false {
			t.Errorf("%s = %v, found %v; want false", key, got, found)
		}
	}
}

func TestEveryBindingNamesAnActionOfFresh(t *testing.T) {
	data, err := os.ReadFile("files/actions.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, list, _ := strings.Cut(string(data), "\n")
	known := strings.Fields(list)
	if !slices.Contains(known, "noop") {
		t.Fatal("the list lacks noop")
	}
	_, c := load(t)
	all := slices.Clone(c.Keybindings)
	for _, m := range c.Maps {
		all = append(all, m.Bindings...)
	}
	for _, b := range all {
		if !slices.Contains(known, b.Action) {
			t.Errorf("%s binds %s, which isn't an action of Fresh", b.spellAll(), b.Action)
		}
	}
}

func TestTheActiveMapResolves(t *testing.T) {
	_, c := load(t)
	m, found := c.Maps[c.Active]
	if !found || m.Inherits != "default" || len(m.Bindings) == 0 {
		t.Fatalf("map %q: %+v", c.Active, m)
	}
}

func TestNoKeyFiresTwoActions(t *testing.T) {
	_, c := load(t)
	seen := map[string]string{}
	for _, b := range c.Maps[c.Active].Bindings {
		mods := slices.Clone(b.Modifiers)
		slices.Sort(mods)
		var seq [][]string
		for _, p := range b.Keys {
			m := slices.Clone(p.Modifiers)
			slices.Sort(m)
			seq = append(seq, append(m, p.Key))
		}
		form, err := json.Marshal([]any{b.When, b.Key, mods, seq})
		if err != nil {
			t.Fatal(err)
		}
		if other, taken := seen[string(form)]; taken {
			t.Errorf("%s binds both %s and %s", form, other, b.Action)
		}
		seen[string(form)] = b.Action
	}
}

func TestNoKeyConflictsWithItsTerminalEquivalent(t *testing.T) {
	_, c := load(t)
	bound := map[string]string{}
	for _, b := range c.Maps[c.Active].Bindings {
		if len(b.Keys) == 0 {
			bound[b.When+" "+spell(b.press)] = b.Action
		}
	}
	for _, when := range []string{"normal", "file_explorer", "terminal"} {
		for _, pair := range [][2]string{
			{"ctrl+/", "ctrl+7"}, {"ctrl+Backspace", "ctrl+h"}, {"ctrl+Space", "ctrl+@"}, {"ctrl+-", "ctrl+_"},
		} {
			one, two := bound[when+" "+pair[0]], bound[when+" "+pair[1]]
			if one != two && one != "" && two != "" {
				t.Errorf("%s: %s is %s, %s is %s", when, pair[0], one, pair[1], two)
			}
		}
	}
	if bound["normal ctrl+h"] != "" || bound["normal ctrl+Backspace"] != "" {
		t.Error("a map's Ctrl+H or Ctrl+Backspace warns at each start, against Fresh's default")
	}
}

func TestAComposeFileTakesItsLanguageID(t *testing.T) {
	loose, _ := load(t)
	langs, _ := loose["languages"].(map[string]any)
	compose, _ := langs["docker-compose"].(map[string]any)
	names, _ := compose["filenames"].([]any)
	for _, file := range []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"} {
		if !slices.Contains(names, any(file)) {
			t.Errorf("docker-compose lacks the file %s", file)
		}
	}
	servers, _ := loose["lsp"].(map[string]any)
	for language, list := range servers {
		for _, entry := range list.([]any) {
			ids := entry.(map[string]any)["language_id_overrides"]
			if (ids != nil) != (language == "docker-compose") {
				t.Errorf("%s: language_id_overrides = %v", language, ids)
			}
		}
	}
}

// byWhen maps "when strokes" to the action of each binding of the active map.
func byWhen(c config) map[string]string {
	out := map[string]string{}
	for _, b := range c.Maps[c.Active].Bindings {
		out[b.When+" "+b.spellAll()] = b.Action
	}
	return out
}

// A key that terminals rewrite arrives as another key. Each form of it binds what the key binds.
func TestEachArrivalIsBound(t *testing.T) {
	_, c := load(t)
	bound := byWhen(c)
	for _, a := range []struct {
		key      string
		arrivals []string
	}{
		{"alt+shift+Space", []string{"alt+Space"}},
	} {
		found := false
		for _, when := range []string{"normal", "file_explorer", "terminal"} {
			action, has := bound[when+" "+a.key]
			if !has {
				continue
			}
			found = true
			for _, other := range a.arrivals {
				if got := bound[when+" "+other]; got != action {
					t.Errorf("%s: %s is %q, but %s is %q", when, a.key, action, other, got)
				}
			}
		}
		if !found {
			t.Errorf("%s binds nothing", a.key)
		}
	}
}

func TestEachShellKeyIsNoopInTheTerminal(t *testing.T) {
	_, c := load(t)
	bound := byWhen(c)
	for _, key := range []string{
		"ctrl+q", "alt+x", "alt+v", "alt+Left", "alt+Right", "alt+e", "alt+f", "alt+g", "alt+h",
		"alt+s", "alt+w", "alt+r", "alt+j", "alt+/", "alt+[", "alt+]", "alt+`", "ctrl+p", "ctrl+;",
		"ctrl+'",
	} {
		if bound["terminal "+key] != "noop" {
			t.Errorf("%s is %q in the terminal; want noop", key, bound["terminal "+key])
		}
	}
	for _, way := range []string{"ctrl+Space", "ctrl+]", "F9", "alt+o"} {
		if bound["terminal "+way] == "noop" {
			t.Errorf("%s is Fresh's way out of its terminal", way)
		}
	}
}
