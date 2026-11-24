package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// serverLock is the lock that `mise lock` writes for the server's env, beside the side's link.
const serverLock = "../guest/mise/mise.guest.lock"

// pinnedTools returns the version that each role's [tools] pins, by tool. It fails the test on a
// tool that two files pin, as mise would keep one of the two with no error.
func pinnedTools(t *testing.T) map[string]string {
	files, err := filepath.Glob(rolesDir + "/*/mise.*.toml")
	if err != nil {
		t.Fatal(err)
	}
	pins, by := map[string]string{}, map[string]string{}
	for _, f := range files {
		text, err := os.ReadFile(f) //nolint:gosec // a role's mise file of the tree, found by the glob
		if err != nil {
			t.Fatal(err)
		}
		var c struct {
			// Tools holds a version, or a table with one.
			Tools map[string]any `toml:"tools"`
		}
		if err := toml.Unmarshal(text, &c); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for tool, v := range c.Tools {
			if other, ok := by[tool]; ok {
				t.Errorf("%s pins %s, which %s pins too", f, tool, other)
			}
			version, _ := v.(string)
			if table, ok := v.(map[string]any); ok {
				version, _ = table["version"].(string)
			}
			if version == "" {
				t.Errorf("%s pins no version of %s: %v", f, tool, v)
			}
			pins[tool], by[tool] = version, f
		}
	}
	return pins
}

// The server's lock holds each tool that a role pins, at its version, and no other. So
// The tree's locked = true installs every pin, and a tool's removal takes its entry too. Each role's own test
// checks its tools' sums.
func TestServerLockHoldsEachPinnedTool(t *testing.T) {
	text, err := os.ReadFile(serverLock)
	if err != nil {
		t.Fatal(err)
	}
	var lock struct {
		Tools map[string][]struct {
			Version string `toml:"version"`
		} `toml:"tools"`
	}
	if err := toml.Unmarshal(text, &lock); err != nil {
		t.Fatal(err)
	}
	pins := pinnedTools(t)
	if len(pins) == 0 {
		t.Fatal("no role pins a tool")
	}
	for tool, version := range pins {
		entries := lock.Tools[tool]
		if len(entries) != 1 || entries[0].Version != version {
			t.Errorf("%s: the lock holds %v, want one entry at %s", tool, entries, version)
		}
	}
	for tool := range lock.Tools {
		if _, ok := pins[tool]; !ok {
			t.Errorf("the lock holds %s, which no role pins", tool)
		}
	}
}
