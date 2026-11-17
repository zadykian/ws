package cmd

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// targets maps each of mise's platforms that ws pins to the release's file for it.
var targets = map[string]string{
	"linux-x64":   "ws-linux-amd64",
	"linux-arm64": "ws-linux-arm64",
	"macos-arm64": "ws-darwin-arm64",
}

var lockSum = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// decodeTOML reads the file at path as TOML, loosely, into a map.
func decodeTOML(t *testing.T, path string) map[string]any {
	text, err := os.ReadFile(path) //nolint:gosec // a file of the tree, named by the test
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := toml.Unmarshal(text, &m); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return m
}

// table returns m[key] as a table, or fails the test where m holds none there.
func table(t *testing.T, m map[string]any, key string) map[string]any {
	v, ok := m[key].(map[string]any)
	if !ok {
		t.Fatalf("no table %q in %v", key, m)
	}
	return v
}

// TestPinLocksEachTarget holds the pin of ws in mise.toml and its checksums in mise.lock together,
// so that a bump leaves no target behind.
func TestPinLocksEachTarget(t *testing.T) {
	tool := table(t, table(t, decodeTOML(t, "../mise.toml"), "tools"), "http:ws")
	version, _ := tool["version"].(string)
	if version == "" {
		t.Fatalf("mise.toml pins no version of http:ws: %v", tool)
	}
	platforms := table(t, tool, "platforms")
	if len(platforms) != len(targets) {
		t.Errorf("mise.toml names %d platforms, want %d: %v", len(platforms), len(targets), platforms)
	}
	for platform, file := range targets {
		url, _ := table(t, platforms, platform)["url"].(string)
		if !strings.HasSuffix(url, "/v{{ version }}/"+file) {
			t.Errorf("mise.toml: %s's url %q does not end /v{{ version }}/%s", platform, url, file)
		}
	}

	locked := table(t, decodeTOML(t, "../mise.lock"), "tools")
	entries, ok := locked["http:ws"].([]any)
	if !ok || len(entries) != 1 {
		t.Fatalf("mise.lock holds %v for http:ws, want one entry", locked["http:ws"])
	}
	entry, _ := entries[0].(map[string]any)
	if entry["version"] != version {
		t.Errorf("mise.lock locks %v, mise.toml pins %s", entry["version"], version)
	}
	for platform, file := range targets {
		p := table(t, entry, "platforms."+platform)
		if sum, _ := p["checksum"].(string); !lockSum.MatchString(sum) {
			t.Errorf("mise.lock: %s's checksum %q is not sha256: and 64 hex digits", platform, sum)
		}
		if url, _ := p["url"].(string); !strings.HasSuffix(url, "/v"+version+"/"+file) {
			t.Errorf("mise.lock: %s's url %q does not end /v%s/%s", platform, url, version, file)
		}
	}
}
