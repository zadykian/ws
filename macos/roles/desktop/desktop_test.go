package desktop_test

import (
	"os"
	"reflect"
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

// TestNamedSettings holds the Dock's and the Finder's named keys. mise ignores a key it does not
// know with a warning, so the keys here are the ones that its tables list.
func TestNamedSettings(t *testing.T) {
	m := load(t).Bootstrap.Settings.Macos
	if want := map[string]any{"tilesize": int64(74), "show_recents": false}; !reflect.DeepEqual(m.Dock, want) {
		t.Errorf("dock = %#v, want %#v", m.Dock, want)
	}
	if want := map[string]any{"preferred_view_style": "column"}; !reflect.DeepEqual(m.Finder, want) {
		t.Errorf("finder = %#v, want %#v", m.Finder, want)
	}
}

// TestDefaultsKeepTheirTypes pins each preference's type, which mise compares with the plist's.
// An integer 0 never equals a real 0.0, so status would differ for good where the type here is
// not the Mac's. TOML's 1.0 is a real, and its 0 an integer.
func TestDefaultsKeepTheirTypes(t *testing.T) {
	want := map[string]map[string]any{
		"com.apple.dock":              {"minimize-to-application": true, "wvous-br-corner": int64(1)},
		"com.apple.finder":            {"NewWindowTarget": "PfHm", "ShowRecentTags": false},
		"com.apple.assistant.support": {"Assistant Enabled": false},
		"com.apple.screencapture": {
			"target": "clipboard", "target-screenshot": "clipboard",
			"target-screenrecording": "file", "style": "selection",
		},
		"com.apple.WindowManager": {
			"EnableStandardClickToShowDesktop": false, "EnableTiledWindowMargins": false,
			"EnableTilingByEdgeDrag": false, "EnableTopTilingByEdgeDrag": false,
			"EnableTilingOptionAccelerator": false,
			"StandardHideWidgets":           true, "StageManagerHideWidgets": true,
		},
		"NSGlobalDomain": {
			"AppleInterfaceStyle":                      "Dark",
			"AppleInterfaceStyleSwitchesAutomatically": false,
			"AppleShowAllExtensions":                   true,
			"AppleShowScrollBars":                      "Always",
			"AppleEnableSwipeNavigateWithScrolls":      false,
			"NSGlassTintAmount":                        float64(1),
			"com.apple.trackpad.scaling":               0.5,
			"com.apple.sound.beep.volume":              float64(0),
			"com.apple.sound.beep.sound":               "/System/Library/Sounds/Tink.aiff",
			"com.apple.sound.uiaudio.enabled":          int64(0),
		},
	}
	got := load(t).Bootstrap.Settings.Macos.Defaults
	for domain, keys := range want {
		for key, value := range keys {
			if g, ok := got[domain][key]; !ok || !reflect.DeepEqual(g, value) {
				t.Errorf("%s %q = %#v (%T), want %#v (%T)", domain, key, g, g, value, value)
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
// where macOS keeps it. mise writes that scope by an entry with host = "current" alone.
func TestScreenSaverIdleTimeIsThisHosts(t *testing.T) {
	entries := load(t).Bootstrap.Settings.Macos.DefaultsEntries
	if len(entries) != 1 {
		t.Fatalf("defaults_entries holds %d entries, want the screen saver's alone", len(entries))
	}
	e := entries[0]
	if e.Domain != "com.apple.screensaver" || e.Key != "idleTime" || e.Host != "current" || e.Value != int64(7200) {
		t.Errorf("the entry is %+v, want idleTime = 7200 of com.apple.screensaver, host current", e)
	}
}
