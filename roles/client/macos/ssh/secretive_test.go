package ssh_test

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// TestSecretivePinIsOneMiseTakes holds the fields that mise's macos-app needs. mise warns and
// skips an entry that lacks one, and the Mac then has no Secretive.
func TestSecretivePinIsOneMiseTakes(t *testing.T) {
	text, err := os.ReadFile("mise.mac.toml")
	if err != nil {
		t.Fatal(err)
	}
	var c struct {
		Bootstrap struct {
			Packages map[string]any `toml:"packages"`
		} `toml:"bootstrap"`
	}
	if err := toml.Unmarshal(text, &c); err != nil {
		t.Fatal(err)
	}
	pin, ok := c.Bootstrap.Packages["macos-app:secretive"].(map[string]any)
	if !ok {
		t.Fatal("mise.mac.toml has no table for macos-app:secretive")
	}
	if v, _ := pin["version"].(string); !regexp.MustCompile(`^\d+\.\d+\.\d+$`).MatchString(v) {
		t.Errorf("version is %q, want one release's number, not latest", pin["version"])
	}
	url, _ := pin["url"].(string)
	if !strings.HasPrefix(url, "https://") || !strings.Contains(url, "{{version}}") {
		t.Errorf("url is %q, want https and {{version}}, so that a release changes the version alone", url)
	}
	if sum, _ := pin["sha256"].(string); !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(sum) {
		t.Errorf("sha256 is %q, want 64 hex digits", pin["sha256"])
	}
	if pin["artifact"] != "Secretive.app" {
		t.Errorf("artifact is %q, want Secretive.app", pin["artifact"])
	}
	if pin["adopt"] != true {
		t.Errorf("adopt is %v, want true: mise takes over the same build that stands", pin["adopt"])
	}
}
