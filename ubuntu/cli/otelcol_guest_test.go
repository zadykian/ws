package cli

import (
	"os"
	"testing"

	"github.com/pelletier/go-toml/v2"

	"github.com/zadykian/ws/ubuntu/values"
)

// miseFile is the part of a role's mise file that these tests read.
type miseFile struct {
	Bootstrap struct {
		Users    map[string]struct{ UID int }
		Groups   map[string]struct{ GID int }
		Packages map[string]any
		Services map[string]any
		Files    map[string]any
	}
}

// readMise parses a role's mise file.
func readMise(t *testing.T, name string) miseFile {
	text, err := os.ReadFile(rolesDir + "/" + name) //nolint:gosec // a role's file, named by the test
	if err != nil {
		t.Fatal(err)
	}
	var f miseFile
	if err := toml.Unmarshal(text, &f); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return f
}

// The profile names the collector's account by number, for the socket's owner. So ws makes the
// account with those ids before the .deb, which keeps an account that it finds.
func TestTheGuestMakesTheCollectorsAccountAtTheProfilesIds(t *testing.T) {
	f := readMise(t, "otelcol/mise.guest.toml")
	user, group := f.Bootstrap.Users["otelcol-contrib"], f.Bootstrap.Groups["otelcol-contrib"]
	if user.UID != values.OtelcolID || group.GID != values.OtelcolID {
		t.Errorf("otelcol-contrib is made with uid %d and gid %d, want %d for both, as the profile's device names them",
			user.UID, group.GID, values.OtelcolID)
	}
}

// A guest's sysfs shows the host's hwmon, so node_exporter, its service and its arguments are the
// host's alone.
func TestNodeExporterIsTheHostsAlone(t *testing.T) {
	const pkg, file = "apt:prometheus-node-exporter", "/etc/default/prometheus-node-exporter"
	guest, host := readMise(t, "otelcol/mise.guest.toml"), readMise(t, "otelcol/mise.host.toml")
	if _, ok := guest.Bootstrap.Packages[pkg]; ok {
		t.Errorf("the guest installs %s", pkg)
	}
	if _, ok := guest.Bootstrap.Services["prometheus-node-exporter"]; ok {
		t.Error("the guest runs node_exporter's service")
	}
	if _, ok := guest.Bootstrap.Files[file]; ok {
		t.Errorf("the guest writes %s", file)
	}
	if _, ok := host.Bootstrap.Packages[pkg]; !ok {
		t.Errorf("the host lacks %s, which its collector scrapes", pkg)
	}
}
