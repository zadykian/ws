package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// filterWords are what a guest's files may not hold in a line that is no comment: a guest filters
// nothing. A host isolates a container, and a cloud VM has its provider's firewall.
var filterWords = []string{"ufw", "firewall", "inet ws", "outbound.nft", "WS_SECURITY_OUTBOUND"}

// TestGuestHoldsNoFilter fails where a role's mise file or spec sets up ufw, a firewall or the
// outbound table, or names the key that turned the table off.
func TestGuestHoldsNoFilter(t *testing.T) {
	var files []string
	for _, name := range append([]string{"units.toml.tmpl"}, roleFiles...) {
		found, err := filepath.Glob(filepath.Join(rolesDir, "*", name))
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, found...)
	}
	for _, f := range files {
		text, err := os.ReadFile(f) //nolint:gosec // a file of the tree, found by the test
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(text), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "#") {
				continue
			}
			for _, w := range filterWords {
				if strings.Contains(line, w) {
					t.Errorf("%s:%d holds %q: a guest holds no filter", f, i+1, w)
				}
			}
		}
	}
	if entries, err := os.ReadDir(filepath.Join(rolesDir, "security", "files")); err == nil {
		for _, e := range entries {
			if strings.Contains(e.Name(), "outbound") || strings.Contains(e.Name(), "resolvers") {
				t.Errorf("security/files/%s is a file of the outbound table", e.Name())
			}
		}
	}
}
