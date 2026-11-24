package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// TestMacFormulaeTakeTheLatest keeps each formula and cask of the Mac's roles at "latest". Homebrew
// publishes a bottle of the current version alone, so a version in the value makes mise skip the
// entry with a warning.
func TestMacFormulaeTakeTheLatest(t *testing.T) {
	files, err := filepath.Glob(macRolesDir + "/*/" + macRoleFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		text, err := os.ReadFile(f) //nolint:gosec // a file of the tree, found by the test
		if err != nil {
			t.Fatal(err)
		}
		var c struct {
			Bootstrap struct {
				Packages map[string]any `toml:"packages"`
			} `toml:"bootstrap"`
		}
		if err := toml.Unmarshal(text, &c); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for name, value := range c.Bootstrap.Packages {
			if strings.HasPrefix(name, "brew:") || strings.HasPrefix(name, "brew-cask:") {
				if value != "latest" {
					t.Errorf("%s: %s = %v, want \"latest\": Homebrew keeps the current version alone", f, name, value)
				}
			}
		}
	}
}
