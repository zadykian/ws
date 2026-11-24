package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

var fullCommit = regexp.MustCompile(`^[0-9a-f]{40}$`)

// TestMacClonesArePinnedToACommit keeps each clone of the Mac's roles at a full commit SHA, with no
// depth. So a run never takes a branch's moving head, and each clone holds the history of its pin.
func TestMacClonesArePinnedToACommit(t *testing.T) {
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
				Repos map[string]map[string]any `toml:"repos"`
			} `toml:"bootstrap"`
		}
		if err := toml.Unmarshal(text, &c); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for path, repo := range c.Bootstrap.Repos {
			if ref, _ := repo["ref"].(string); !fullCommit.MatchString(ref) {
				t.Errorf("%s: %s has ref %q, want a full commit SHA", f, path, repo["ref"])
			}
			if _, ok := repo["depth"]; ok {
				t.Errorf("%s: %s has a depth, want a full clone", f, path)
			}
		}
	}
}
