package apps_test

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// appOf names the cask that installs each app that the Dock pins or that opens at login, from
// /Applications. Safari is the system's and has none.
var appOf = map[string]string{
	"Safari.app":           "",
	"Telegram Desktop.app": "telegram-desktop",
	"Discord.app":          "discord",
	"Claude.app":           "claude",
	"Zed.app":              "zed",
	"Ghostty.app":          "ghostty",
	"Yandex Telemost.app":  "yandextelemost",
	"Rectangle Pro.app":    "rectangle-pro",
	"Stats.app":            "stats",
}

type appsFile struct {
	Bootstrap struct {
		Packages map[string]any `toml:"packages"`
		Settings struct {
			MacOS struct {
				Dock struct {
					Apps []string `toml:"apps"`
				} `toml:"dock"`
				LoginItems struct {
					Exclusive bool     `toml:"exclusive"`
					Apps      []string `toml:"apps"`
				} `toml:"login_items"`
				Defaults map[string]map[string]any `toml:"defaults"`
			} `toml:"macos"`
		} `toml:"settings"`
	} `toml:"bootstrap"`
}

func read(t *testing.T, name string) appsFile {
	text, err := os.ReadFile(name) //nolint:gosec // a file of the tree, named by the test
	if err != nil {
		t.Fatal(err)
	}
	var f appsFile
	if err := toml.Unmarshal(text, &f); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return f
}

// casksOfRoles returns each cask that a role of the Mac installs, apps' or not.
func casksOfRoles(t *testing.T) []string {
	var casks []string
	for _, name := range []string{"mise.macos.toml", "mise.apps.toml"} {
		files, err := filepath.Glob("../*/" + name)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			for pkg := range read(t, f).Bootstrap.Packages {
				if cask, ok := strings.CutPrefix(pkg, "brew-cask:"); ok {
					casks = append(casks, cask)
				}
			}
		}
	}
	return casks
}

// missingApps lists the apps that the setting names but neither the system nor a cask of the roles
// installs, each with what is wrong.
func missingApps(t *testing.T, setting string, apps []string) []string {
	casks := casksOfRoles(t)
	var problems []string
	for _, app := range apps {
		if strings.HasPrefix(app, "/System/Applications/") {
			continue
		}
		name, ok := strings.CutPrefix(app, "/Applications/")
		cask, known := appOf[name]
		switch {
		case !ok || !known:
			problems = append(problems, setting+" names "+app+", which this test does not know: add its cask to appOf")
		case cask != "" && !slices.Contains(casks, cask):
			problems = append(problems, setting+" names "+app+", but no role installs the cask "+cask)
		}
	}
	return problems
}

// TestEachDockAppIsInstalledBeforeTheDockIsWritten holds the Dock's apps to ones that exist: mise
// fails a changed layout where an app is missing, which only a Mac would show.
func TestEachDockAppIsInstalledBeforeTheDockIsWritten(t *testing.T) {
	for _, p := range missingApps(t, "the Dock", read(t, "mise.apps.toml").Bootstrap.Settings.MacOS.Dock.Apps) {
		t.Error(p)
	}
}

// TestEachLoginItemIsAnInstalledApp holds the login items to apps that the system or a cask
// installs, as the list is the maintainer's seven and no other.
func TestEachLoginItemIsAnInstalledApp(t *testing.T) {
	items := read(t, "mise.apps.toml").Bootstrap.Settings.MacOS.LoginItems
	if len(items.Apps) != 7 {
		t.Errorf("login items = %q, want seven apps", items.Apps)
	}
	for _, p := range missingApps(t, "the login items", items.Apps) {
		t.Error(p)
	}
}

// TestTheLoginItemsAreExclusive keeps the list ws's alone: an item added by hand goes at the next
// run, and a missing key would leave it.
func TestTheLoginItemsAreExclusive(t *testing.T) {
	if !read(t, "mise.apps.toml").Bootstrap.Settings.MacOS.LoginItems.Exclusive {
		t.Error("login_items.exclusive is not true: an item added by hand would stay")
	}
}

// TestTheAppsHoldNoUnitAndNoFormula keeps the file to casks: its formulae and units belong in a
// role's own file, which the first run loads.
func TestTheAppsHoldNoUnitAndNoFormula(t *testing.T) {
	for pkg := range read(t, "mise.apps.toml").Bootstrap.Packages {
		if !strings.HasPrefix(pkg, "brew-cask:") {
			t.Errorf("%s is no cask: mise.apps.toml holds casks alone", pkg)
		}
	}
}

// statsDomain is Stats's bundle ID. Stats runs outside the sandbox, so its preferences sit in
// ~/Library/Preferences, where mise writes them.
const statsDomain = "eu.exelban.Stats"

// statsKeys names each key of Stats's that the role writes, by its value's type in the
// maintainer's export. Stats reads a value of another type as missing, and takes its default.
var statsKeys = map[string][]string{
	"array": {
		"CombinedModules_modifierKeys",
	},
	"bool": {
		"Battery_battery_color", "Battery_battery_xlSize", "Battery_state", "Bluetooth_state",
		"CPU_bar_chart_box", "CPU_bar_chart_frame", "CPU_bar_chart_label", "CPU_clustersGroup",
		"CPU_hyperhreading", "CPU_line_chart_box", "CPU_line_chart_frame", "CPU_line_chart_label",
		"CPU_line_chart_value", "CPU_splitValue", "CPU_state", "CPU_usagePerCore", "Clock_state",
		"CombinedModules", "CombinedModules_alternateDetails", "CombinedModules_separator",
		"Disk_state", "GPU_state", "Network_oneView", "Network_speed_monochrome", "Network_speed_value",
		"Network_state", "RAM_bar_chart_label", "RAM_combinedProcesses", "RAM_splitValue", "RAM_state",
		"Sensors_state", "sensor_CPU Power", "sensor_Hottest CPU", "sensor_Hottest GPU", "sensor_PPBR",
		"sensor_TH0x", "sensor_TaLP", "sensor_Total System Consumption", "setupProcess",
	},
	"int": {
		"Battery_position", "Bluetooth_position", "CPU_barChart_position", "CPU_icon_position",
		"CPU_label_position", "CPU_lineChart_position", "CPU_mini_position", "CPU_pieChart_position",
		"CPU_position", "CPU_tachometer_position", "CPU_updateInterval", "CPU_updateTopInterval",
		"Clock_icon_position", "Clock_label_position", "Clock_position", "Clock_stack_position",
		"Disk_barChart_position", "Disk_icon_position", "Disk_label_position", "Disk_memory_position",
		"Disk_mini_position", "Disk_networkChart_position", "Disk_pieChart_position", "Disk_position",
		"Disk_speed_position", "Disk_text_position", "GPU_barChart_position", "GPU_icon_position",
		"GPU_label_position", "GPU_lineChart_position", "GPU_mini_position", "GPU_position",
		"GPU_tachometer_position", "GPU_text_position", "GPU_updateInterval", "Network_icon_position",
		"Network_label_position", "Network_networkChart_position", "Network_position",
		"Network_speed_position", "Network_state_position", "Network_text_position",
		"Network_updateICMPInterval", "RAM_barChart_position", "RAM_icon_position",
		"RAM_label_position", "RAM_lineChart_position", "RAM_memory_position", "RAM_mini_position",
		"RAM_pieChart_position", "RAM_position", "RAM_state_position", "RAM_tachometer_position",
		"RAM_text_position", "Remote_position", "Sensors_barChart_position", "Sensors_icon_position",
		"Sensors_label_position", "Sensors_mini_position", "Sensors_position", "Sensors_stack_position",
	},
	"string": {
		"Battery_battery_additional", "CPU_bar_chart_color", "CPU_line_chart_color", "CPU_widget",
		"Clock_widget", "CombinedModules_display", "Disk_widget", "GPU_widget", "Network_speedUnit",
		"Network_speed_icon", "Network_speed_valueColor", "Network_widget", "RAM_bar_chart_color",
		"RAM_widget", "Sensors_widget",
	},
}

// statsState holds the keys of the export that Stats or AppKit writes: times, its version, records
// of a step done, its window's frames and its panels' last folder. A run would write each back
// after Stats changed it.
var statsState = []string{
	"support_ts", "updater_check_ts", "version", "runAtLoginInitialized",
	"remote_tokens_migrated_to_keychain", "NSWindow Frame eu.exelban.Stats.Settings.WindowFrame",
	"NSToolbar Configuration eu.exelban.Stats.Settings.Toolbar", "NSOSPLastRootDirectory",
}

func stats(t *testing.T) map[string]any {
	return read(t, "mise.apps.toml").Bootstrap.Settings.MacOS.Defaults[statsDomain]
}

// kindOf names a value's type as statsKeys does. go-toml reads TOML's integer as int64, and its
// array as []any.
func kindOf(value any) string {
	switch value.(type) {
	case []any:
		return "array"
	case bool:
		return "bool"
	case int64:
		return "int"
	case string:
		return "string"
	}
	return fmt.Sprintf("%T", value)
}

// TestStatsKeysKeepTheExportsTypes holds each key of Stats's to its type in the export. mise
// compares the type too, so a key of another type would stay different for good.
func TestStatsKeysKeepTheExportsTypes(t *testing.T) {
	got := stats(t)
	pinned := map[string]bool{}
	for kind, keys := range statsKeys {
		for _, key := range keys {
			pinned[key] = true
			value, ok := got[key]
			switch {
			case !ok:
				t.Errorf("Stats's %s is not set", key)
			case kindOf(value) != kind:
				t.Errorf("Stats's %s is %s, want %s as in the export", key, kindOf(value), kind)
			}
		}
	}
	for key := range got {
		if !pinned[key] {
			t.Errorf("Stats's %s is set, and this test does not pin its type", key)
		}
	}
}

// TestStatsHoldsNoStateKey keeps out the keys that Stats writes itself, which would leave the
// plan different after each start of Stats.
func TestStatsHoldsNoStateKey(t *testing.T) {
	got := stats(t)
	for _, key := range statsState {
		if _, ok := got[key]; ok {
			t.Errorf("Stats's %q is set: Stats writes it itself", key)
		}
	}
}

// TestStatsSkipsItsSetupWindow holds setupProcess true. Without it Stats opens its setup window,
// whose first page sets each module's state and widget again.
func TestStatsSkipsItsSetupWindow(t *testing.T) {
	if stats(t)["setupProcess"] != true {
		t.Error("Stats's setupProcess is not true: Stats would show its setup window")
	}
}
