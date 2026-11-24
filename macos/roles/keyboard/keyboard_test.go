package keyboard_test

import (
	"os"
	"reflect"
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
