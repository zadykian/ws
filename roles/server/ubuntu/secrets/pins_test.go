package secrets_test

import (
	"os"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// fnoxSums are GitHub's digests of fnox 1.50.0's Linux assets, by mise's platform and the asset's
// target.
var fnoxSums = map[string]struct{ target, sha string }{
	"linux-x64":   {"x86_64-unknown-linux-gnu", "f08da606e120c676fc3a4f3def970a4ade82e3857841a3961afbdd337a88e30b"},
	"linux-arm64": {"aarch64-unknown-linux-gnu", "605b3f5fdd8cf4806d7be63ca4ef52c6fa26d86ac7c97a04a3263daf990233dd"},
}

// The lock's fnox entry is at the version that [tools] pins, and downloads each platform's asset of
// that release against GitHub's digest. A bump moves the pin, the lock and these sums together.
func TestFnoxLockHoldsGitHubsDigests(t *testing.T) {
	text, err := os.ReadFile("mise.server.toml")
	if err != nil {
		t.Fatal(err)
	}
	var c struct {
		Tools map[string]string `toml:"tools"`
	}
	if err := toml.Unmarshal(text, &c); err != nil {
		t.Fatal(err)
	}
	version := c.Tools["fnox"]
	if version != "1.50.0" {
		t.Fatalf("[tools] pins fnox %q, want the version of the sums, 1.50.0", version)
	}
	text, err = os.ReadFile("../../mise/mise.server.lock")
	if err != nil {
		t.Fatal(err)
	}
	var lock struct {
		Tools map[string][]map[string]any `toml:"tools"`
	}
	if err := toml.Unmarshal(text, &lock); err != nil {
		t.Fatal(err)
	}
	entries := lock.Tools["fnox"]
	if len(entries) != 1 || entries[0]["version"] != version {
		t.Fatalf("the lock's fnox is %v, want one entry at %s", entries, version)
	}
	for platform, want := range fnoxSums {
		p, _ := entries[0]["platforms."+platform].(map[string]any)
		url := "https://github.com/jdx/fnox/releases/download/v" + version + "/fnox-" + want.target + ".tar.gz"
		if p["checksum"] != "sha256:"+want.sha || p["url"] != url {
			t.Errorf("%s: the lock holds %v, want sha256:%s from %s", platform, p, want.sha, url)
		}
	}
}
