package keyboard_test

import (
	"fmt"
	"os"
	"slices"
	"strings"
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

// TestTheLanguageIndicatorIsAString holds the indicator's value to a string, the type that mise
// compares with the one the Mac holds.
func TestTheLanguageIndicatorIsAString(t *testing.T) {
	got := load(t).Bootstrap.Settings.Macos.Defaults["NSGlobalDomain"]
	if want := "dict{TSMLanguageIndicatorEnabled:string}"; shape(got) != want {
		t.Errorf("NSGlobalDomain = %s, want %s", shape(got), want)
	}
}

// TestCapsLockIsControl holds the mapping to the current host's domain, where macOS keeps one
// per keyboard, and its two codes to integers.
func TestCapsLockIsControl(t *testing.T) {
	const key = "com.apple.keyboard.modifiermapping.0-0-0"
	for _, e := range load(t).Bootstrap.Settings.Macos.DefaultsEntries {
		if e.Key != key {
			continue
		}
		want := "array[dict{HIDKeyboardModifierMappingDst:int,HIDKeyboardModifierMappingSrc:int}]"
		if e.Domain != "NSGlobalDomain" || e.Host != "current" || shape(e.Value) != want {
			t.Errorf("the mapping is %+v, want NSGlobalDomain, host current, %s", e, want)
		}
		return
	}
	t.Errorf("no entry for %s", key)
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

// TestTheHotkeysThatStayOn holds each one's whole value to the type standard. mise writes a path
// into the dictionary, so the IDs that macOS adds stay, and a value that holds a key more or less
// than this reads as different.
func TestTheHotkeysThatStayOn(t *testing.T) {
	const want = "dict{enabled:bool,value:dict{parameters:array[int,int,int],type:string}}"
	stay := 0
	for id, e := range hotkeys(t) {
		if len(e.Path) != 1 {
			continue
		}
		stay++
		if shape(e.Value) != want {
			t.Errorf("hotkey %s = %s, want %s", id, shape(e.Value), want)
		}
	}
	if stay == 0 {
		t.Error("no hotkey stays on")
	}
}

// TestTheHotkeysThatGoOff holds each other ID to a path of its `enabled` key alone, set to a
// boolean. That keeps the hotkey's value, so that it comes back on with the key it had, and an ID
// that macOS lacks gets a dictionary with `enabled` alone.
func TestTheHotkeysThatGoOff(t *testing.T) {
	off := 0
	for id, e := range hotkeys(t) {
		switch {
		case len(e.Path) == 1:
		case len(e.Path) != 2 || e.Path[1] != "enabled" || shape(e.Value) != "bool":
			t.Errorf("hotkey %s = %+v, want path [%s enabled] and a boolean", id, e, id)
		default:
			off++
		}
	}
	if off == 0 {
		t.Error("no hotkey goes off")
	}
}

// shape names a value's type, and each element's and key's below it, so that a test compares what
// mise compares and not the value. An integer and a real differ, as TOML's 1 and 1.0 do.
func shape(value any) string {
	switch v := value.(type) {
	case []any:
		parts := make([]string, len(v))
		for i, e := range v {
			parts[i] = shape(e)
		}
		return "array[" + strings.Join(parts, ",") + "]"
	case map[string]any:
		parts := make([]string, 0, len(v))
		for k, e := range v {
			parts = append(parts, k+":"+shape(e))
		}
		slices.Sort(parts)
		return "dict{" + strings.Join(parts, ",") + "}"
	case bool:
		return "bool"
	case int64:
		return "int"
	case float64:
		return "real"
	case string:
		return "string"
	}
	return fmt.Sprintf("%T", value)
}

// TestTheInputSourcesAreTheLayoutsAlone holds the two arrays to their type. mise compares each
// whole, by element, in order, and by type: an ID is an integer, as macOS stores it.
func TestTheInputSourcesAreTheLayoutsAlone(t *testing.T) {
	const layout = "dict{InputSourceKind:string,KeyboardLayout ID:int,KeyboardLayout Name:string}"
	const method = "dict{Bundle ID:string,InputSourceKind:string}"
	got := load(t).Bootstrap.Settings.Macos.Defaults
	want := map[string]map[string]string{
		"com.apple.inputsources": {"AppleEnabledThirdPartyInputSources": "array[" + layout + "," + layout + "]"},
		"com.apple.HIToolbox": {"AppleEnabledInputSources": "array[" + layout + "," + layout + "," +
			method + "," + method + "]"},
	}
	for domain, keys := range want {
		for key, kind := range keys {
			if g := shape(got[domain][key]); g != kind {
				t.Errorf("%s %s = %s, want %s", domain, key, g, kind)
			}
		}
	}
}
