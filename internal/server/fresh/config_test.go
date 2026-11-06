package fresh

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/host"
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
	data, err := host.Render(templates, "files/config.json.tmpl", struct{ Home string }{home})
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
