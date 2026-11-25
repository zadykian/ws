package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// tableHeader matches a spec's table header, such as [state.docker-dns.ufw] or [[service.x.file]],
// and captures the unit it belongs to, as kind.name.
var tableHeader = regexp.MustCompile(`^\[\[?([a-z]+\.[a-z0-9-]+)[.\]]`)

// hostOnlyUnits returns the units of a spec that run on the host alone, as kind.name: those whose
// table holds when = { side = "host" }.
func hostOnlyUnits(lines []string) map[string]bool {
	units, unit := map[string]bool{}, ""
	for _, line := range lines {
		if m := tableHeader.FindStringSubmatch(line); m != nil {
			unit = m[1]
		}
		if unit != "" && strings.Contains(line, `when = { side = "host" }`) {
			units[unit] = true
		}
	}
	return units
}

// filterWords are what a guest's files may not hold in a line that is no comment: a guest filters
// nothing. A host isolates a container, and a cloud VM has its provider's firewall.
var filterWords = []string{"ufw", "firewall", "inet ws", "outbound.nft", "WS_SECURITY_OUTBOUND"}

// TestGuestHoldsNoFilter fails where a role's guest file or spec sets up ufw, a firewall or the
// outbound table, or names the key that turned the table off. A unit of a spec that runs on the
// host alone may hold ufw: the host's roles hold it, and the guest drops that unit as it loads.
func TestGuestHoldsNoFilter(t *testing.T) {
	var files []string
	for _, name := range []string{"units.toml.tmpl", "mise.guest.toml", "mise.secrets.toml"} {
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
		lines := strings.Split(string(text), "\n")
		hostOnly, unit := hostOnlyUnits(lines), ""
		for i, line := range lines {
			if m := tableHeader.FindStringSubmatch(line); m != nil {
				unit = m[1]
			}
			if strings.HasPrefix(strings.TrimSpace(line), "#") || hostOnly[unit] {
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
