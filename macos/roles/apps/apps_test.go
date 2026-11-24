package apps_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// appOf names the cask that installs each app the Dock pins from /Applications. Safari is the
// system's and has none.
var appOf = map[string]string{
	"Safari.app":           "",
	"Telegram Desktop.app": "telegram-desktop",
	"Discord.app":          "discord",
	"Claude.app":           "claude",
	"Zed.app":              "zed",
	"Ghostty.app":          "ghostty",
	"Yandex Telemost.app":  "yandextelemost",
}

type appsFile struct {
	Bootstrap struct {
		Packages map[string]any `toml:"packages"`
		Settings struct {
			MacOS struct {
				Dock struct {
					Apps []string `toml:"apps"`
				} `toml:"dock"`
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

// TestEachDockAppIsInstalledBeforeTheDockIsWritten holds the Dock's apps to ones that exist: mise
// fails a changed layout where an app is missing, which only a Mac would show.
func TestEachDockAppIsInstalledBeforeTheDockIsWritten(t *testing.T) {
	casks := casksOfRoles(t)
	for _, app := range read(t, "mise.apps.toml").Bootstrap.Settings.MacOS.Dock.Apps {
		if strings.HasPrefix(app, "/System/Applications/") {
			continue
		}
		name, ok := strings.CutPrefix(app, "/Applications/")
		cask, known := appOf[name]
		switch {
		case !ok || !known:
			t.Errorf("the Dock pins %s, which this test does not know: add its cask to appOf", app)
		case cask != "" && !slices.Contains(casks, cask):
			t.Errorf("the Dock pins %s, but no role installs the cask %s", app, cask)
		}
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
