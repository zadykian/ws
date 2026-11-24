package secrets_test

import (
	"crypto/sha256"
	"encoding/base64"
	"os"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// githubFingerprint is the fingerprint of GitHub's ed25519 host key, as docs.github.com's
// "GitHub's SSH key fingerprints" gives it.
const githubFingerprint = "SHA256:+DiY3wvvV6TuJJhbpZisF/zLDA0zPMSvHdkr4UvCOqU"

// The secrets' fetch trusts github.com by the one key that github_known_hosts pins, so the key
// must be the one GitHub publishes. ssh's fingerprint is the SHA-256 of the key's blob, in base64
// with no padding.
func TestGitHubHostKeyMatchesItsFingerprint(t *testing.T) {
	text, err := os.ReadFile("github_known_hosts")
	if err != nil {
		t.Fatal(err)
	}
	var keys [][]string
	for line := range strings.Lines(string(text)) {
		if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "#") {
			keys = append(keys, strings.Fields(line))
		}
	}
	if len(keys) != 1 {
		t.Fatalf("github_known_hosts holds %d keys, want GitHub's ed25519 key alone", len(keys))
	}
	key := keys[0]
	if len(key) != 3 || key[0] != "github.com" || key[1] != "ssh-ed25519" {
		t.Fatalf("the key's line is %q, want github.com ssh-ed25519 and the key", key)
	}
	blob, err := base64.StdEncoding.DecodeString(key[2])
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(blob)
	if got := "SHA256:" + base64.RawStdEncoding.EncodeToString(sum[:]); got != githubFingerprint {
		t.Errorf("the pinned key's fingerprint is %s, want GitHub's %s", got, githubFingerprint)
	}
}

// fnoxSums are GitHub's digests of fnox 1.50.0's Linux assets, by mise's platform and the asset's
// target.
var fnoxSums = map[string]struct{ target, sha string }{
	"linux-x64":   {"x86_64-unknown-linux-gnu", "f08da606e120c676fc3a4f3def970a4ade82e3857841a3961afbdd337a88e30b"},
	"linux-arm64": {"aarch64-unknown-linux-gnu", "605b3f5fdd8cf4806d7be63ca4ef52c6fa26d86ac7c97a04a3263daf990233dd"},
}

// fnoxTool is fnox's name in [tools] and the lock: mise's github backend, which installs from the
// lock's url with no call to GitHub's API.
const fnoxTool = "github:jdx/fnox"

// The lock's fnox entry is at the version that [tools] pins, and downloads each platform's asset of
// that release against GitHub's digest. A bump moves the pin, the lock and these sums together.
func TestFnoxLockHoldsGitHubsDigests(t *testing.T) {
	text, err := os.ReadFile("mise.guest.toml")
	if err != nil {
		t.Fatal(err)
	}
	var c struct {
		Tools map[string]string `toml:"tools"`
	}
	if err := toml.Unmarshal(text, &c); err != nil {
		t.Fatal(err)
	}
	version := c.Tools[fnoxTool]
	if len(c.Tools) != 1 || version != "1.50.0" {
		t.Fatalf("[tools] pins %v, want %s alone at the version of the sums, 1.50.0", c.Tools, fnoxTool)
	}
	text, err = os.ReadFile("../../guest/mise/mise.guest.lock")
	if err != nil {
		t.Fatal(err)
	}
	var lock struct {
		Tools map[string][]map[string]any `toml:"tools"`
	}
	if err := toml.Unmarshal(text, &lock); err != nil {
		t.Fatal(err)
	}
	entries := lock.Tools[fnoxTool]
	if len(entries) != 1 || entries[0]["version"] != version || entries[0]["backend"] != fnoxTool {
		t.Fatalf("the lock's %s is %v, want one entry at %s", fnoxTool, entries, version)
	}
	if stale, ok := lock.Tools["fnox"]; ok {
		t.Errorf("the lock holds an entry of the registry's fnox, %v, which the pin no longer names", stale)
	}
	for platform, want := range fnoxSums {
		p, _ := entries[0]["platforms."+platform].(map[string]any)
		url := "https://github.com/jdx/fnox/releases/download/v" + version + "/fnox-" + want.target + ".tar.gz"
		if p["checksum"] != "sha256:"+want.sha || p["url"] != url {
			t.Errorf("%s: the lock holds %v, want sha256:%s from %s", platform, p, want.sha, url)
		}
	}
}
