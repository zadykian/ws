package keyboard_test

import (
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// entry is one [[defaults_entries]] of the role's mise file.
type entry struct {
	Domain string   `toml:"domain"`
	Key    string   `toml:"key"`
	Host   string   `toml:"host"`
	Path   []string `toml:"path"`
	Value  any      `toml:"value"`
}

// settings is the part of the role's mise file that the tests read.
type settings struct {
	Bootstrap struct {
		Settings struct {
			Macos struct {
				Defaults        map[string]map[string]any `toml:"defaults"`
				DefaultsEntries []entry                   `toml:"defaults_entries"`
			} `toml:"macos"`
		} `toml:"settings"`
	} `toml:"bootstrap"`
}

func load(t *testing.T) settings {
	text, err := os.ReadFile("mise.macos.toml")
	if err != nil {
		t.Fatal(err)
	}
	var s settings
	if err := toml.Unmarshal(text, &s); err != nil {
		t.Fatal(err)
	}
	return s
}

// TestTheLanguageIndicatorIsAString holds the indicator's value to the string "0", the type that
// mise compares with the one the Mac holds.
func TestTheLanguageIndicatorIsAString(t *testing.T) {
	got := load(t).Bootstrap.Settings.Macos.Defaults["NSGlobalDomain"]
	if want := map[string]any{"TSMLanguageIndicatorEnabled": "0"}; !reflect.DeepEqual(got, want) {
		t.Errorf("NSGlobalDomain = %#v, want %#v", got, want)
	}
}

// TestCapsLockIsControl holds the mapping to the current host's domain, where macOS keeps one
// per keyboard, and its two codes to integers.
func TestCapsLockIsControl(t *testing.T) {
	want := entry{
		Domain: "NSGlobalDomain",
		Key:    "com.apple.keyboard.modifiermapping.0-0-0",
		Host:   "current",
		Value: []any{map[string]any{
			"HIDKeyboardModifierMappingSrc": int64(30064771129),
			"HIDKeyboardModifierMappingDst": int64(30064771300),
		}},
	}
	for _, e := range load(t).Bootstrap.Settings.Macos.DefaultsEntries {
		if e.Key == want.Key {
			if !reflect.DeepEqual(e, want) {
				t.Errorf("the mapping is %+v, want %+v", e, want)
			}
			return
		}
	}
	t.Errorf("no entry for %s", want.Key)
}

// hotkeys maps each hotkey's ID to its entry, for the entries that patch the shortcuts'
// dictionary. A path of the ID alone replaces that hotkey's whole value.
func hotkeys(t *testing.T) map[string]entry {
	got := map[string]entry{}
	for _, e := range load(t).Bootstrap.Settings.Macos.DefaultsEntries {
		if e.Domain != "com.apple.symbolichotkeys" {
			continue
		}
		if e.Key != "AppleSymbolicHotKeys" || e.Host != "" || len(e.Path) == 0 {
			t.Errorf("the entry %+v is not one hotkey's path in AppleSymbolicHotKeys", e)
			continue
		}
		if _, ok := got[e.Path[0]]; ok {
			t.Errorf("hotkey %s has two entries", e.Path[0])
		}
		got[e.Path[0]] = e
	}
	return got
}

// TestTheHotkeysThatStayOn holds each one's whole value, of the type standard. mise writes a path
// into the dictionary, so the IDs that macOS adds stay, and a value that holds a key more or less
// than this reads as different.
func TestTheHotkeysThatStayOn(t *testing.T) {
	on := func(code, key, mask int64) entry {
		return entry{
			Domain: "com.apple.symbolichotkeys",
			Key:    "AppleSymbolicHotKeys",
			Value: map[string]any{
				"enabled": true,
				"value": map[string]any{
					"parameters": []any{code, key, mask},
					"type":       "standard",
				},
			},
		}
	}
	want := map[string]entry{
		"61":  on(32, 49, 1048576),
		"64":  on(32, 49, 524288),
		"160": on(108, 37, 1048576),
	}
	got := hotkeys(t)
	for id, w := range want {
		w.Path = []string{id}
		if !reflect.DeepEqual(got[id], w) {
			t.Errorf("hotkey %s = %+v, want %+v", id, got[id], w)
		}
	}
}

// TestTheHotkeysThatGoOff holds each ID to a path of its `enabled` key alone. That keeps the
// hotkey's value, so that it comes back on with the key it had, and an ID that macOS lacks gets a
// dictionary with `enabled` alone. 34 and 98 stay as macOS has them, and no ID is on and off.
func TestTheHotkeysThatGoOff(t *testing.T) {
	var want []string
	for _, r := range [][2]int{
		{7, 13}, {21, 21}, {25, 27}, {32, 33}, {36, 36}, {52, 54}, {57, 57}, {59, 60}, {65, 65},
		{159, 159}, {162, 162}, {164, 164}, {175, 175}, {190, 190}, {215, 219}, {222, 233},
		{235, 235}, {237, 251}, {256, 258}, {260, 260},
	} {
		for id := r[0]; id <= r[1]; id++ {
			want = append(want, strconv.Itoa(id))
		}
	}
	got := map[string]entry{}
	for id, e := range hotkeys(t) {
		if len(e.Path) == 2 {
			got[id] = e
		}
	}
	if len(got) != len(want) {
		t.Errorf("%d hotkeys go off, want %d", len(got), len(want))
	}
	for _, id := range want {
		e, ok := got[id]
		if !ok {
			t.Errorf("hotkey %s has no entry that turns it off", id)
		} else if !reflect.DeepEqual(e.Path, []string{id, "enabled"}) || e.Value != false {
			t.Errorf("hotkey %s = %+v, want path [%s enabled] and false", id, e, id)
		}
	}
	// The three that stay on have a path of the ID alone: no other entry, and none deeper.
	all := hotkeys(t)
	if len(all) != len(want)+3 {
		t.Errorf("%d hotkeys have an entry, want %d", len(all), len(want)+3)
	}
	for id, e := range all {
		if len(e.Path) > 2 {
			t.Errorf("hotkey %s has the path %v, deeper than its enabled key", id, e.Path)
		}
	}
	for _, id := range []string{"34", "98"} {
		if _, ok := all[id]; ok {
			t.Errorf("hotkey %s has an entry, and none is wanted", id)
		}
	}
}
