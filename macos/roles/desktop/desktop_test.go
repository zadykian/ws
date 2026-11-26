package desktop_test

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// settings is the part of the role's mise file that the tests read.
type settings struct {
	Bootstrap struct {
		Settings struct {
			Macos struct {
				Dock            map[string]any            `toml:"dock"`
				Finder          map[string]any            `toml:"finder"`
				Defaults        map[string]map[string]any `toml:"defaults"`
				DefaultsEntries []struct {
					Domain string `toml:"domain"`
					Key    string `toml:"key"`
					Host   string `toml:"host"`
					Value  any    `toml:"value"`
				} `toml:"defaults_entries"`
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

// TestNamedSettings holds the Dock's and the Finder's named keys, each to its type. mise ignores
// a key it does not know with a warning, so the keys here are the ones that its tables list.
func TestNamedSettings(t *testing.T) {
	m := load(t).Bootstrap.Settings.Macos
	if got, want := shape(m.Dock), "dict{show_recents:bool,tilesize:int}"; got != want {
		t.Errorf("dock = %s, want %s", got, want)
	}
	if got, want := shape(m.Finder), "dict{preferred_view_style:string}"; got != want {
		t.Errorf("finder = %s, want %s", got, want)
	}
}

// TestDefaultsKeepTheirTypes pins each preference's type, which mise compares with the plist's.
// An integer 0 never equals a real 0.0, so status would differ for good where the type here is
// not the Mac's. TOML's 1.0 is a real, and its 0 an integer.
func TestDefaultsKeepTheirTypes(t *testing.T) {
	want := map[string]map[string]string{
		"com.apple.dock":              {"minimize-to-application": "bool", "wvous-br-corner": "int"},
		"com.apple.finder":            {"NewWindowTarget": "string", "ShowRecentTags": "bool"},
		"com.apple.assistant.support": {"Assistant Enabled": "bool"},
		"com.apple.screencapture": {
			"target": "string", "target-screenshot": "string",
			"target-screenrecording": "string", "style": "string",
		},
		"com.apple.WindowManager": {
			"EnableStandardClickToShowDesktop": "bool", "EnableTiledWindowMargins": "bool",
			"EnableTilingByEdgeDrag": "bool", "EnableTopTilingByEdgeDrag": "bool",
			"EnableTilingOptionAccelerator": "bool",
			"StandardHideWidgets":           "bool", "StageManagerHideWidgets": "bool",
		},
		"NSGlobalDomain": {
			"AppleInterfaceStyle":                      "string",
			"AppleInterfaceStyleSwitchesAutomatically": "bool",
			"AppleShowAllExtensions":                   "bool",
			"AppleShowScrollBars":                      "string",
			"AppleEnableSwipeNavigateWithScrolls":      "bool",
			"NSGlassTintAmount":                        "real",
			"com.apple.trackpad.scaling":               "real",
			"com.apple.sound.beep.volume":              "real",
			"com.apple.sound.beep.sound":               "string",
			"com.apple.sound.uiaudio.enabled":          "int",
		},
	}
	got := load(t).Bootstrap.Settings.Macos.Defaults
	for domain, keys := range want {
		for key, kind := range keys {
			value, ok := got[domain][key]
			if !ok || shape(value) != kind {
				t.Errorf("%s %q is %s, want %s", domain, key, shape(value), kind)
			}
		}
	}
	for domain, keys := range got {
		for key := range keys {
			if _, ok := want[domain][key]; !ok {
				t.Errorf("%s %q is set, and this test does not pin its type", domain, key)
			}
		}
	}
}

// TestScreenSaverIdleTimeIsThisHosts holds the screen saver's delay in the current host's domain,
// where macOS keeps it, as an integer. mise writes that scope by an entry with host = "current" alone.
func TestScreenSaverIdleTimeIsThisHosts(t *testing.T) {
	entries := load(t).Bootstrap.Settings.Macos.DefaultsEntries
	if len(entries) != 1 {
		t.Fatalf("defaults_entries holds %d entries, want the screen saver's alone", len(entries))
	}
	e := entries[0]
	if e.Domain != "com.apple.screensaver" || e.Key != "idleTime" || e.Host != "current" || shape(e.Value) != "int" {
		t.Errorf("the entry is %+v, want idleTime of com.apple.screensaver, host current, an integer", e)
	}
}
