package apps_test

import (
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
