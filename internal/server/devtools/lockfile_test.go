package devtools

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTheLockfilePinsEachPackageOfPackageJSON(t *testing.T) {
	var pkg struct{ Dependencies map[string]string }
	var lock struct {
		LockfileVersion int
		Packages        map[string]struct{ Version, Integrity, Resolved string }
	}
	for name, into := range map[string]any{"servers/package.json": &pkg, "servers/package-lock.json": &lock} {
		data, err := serversLock.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, into); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if len(pkg.Dependencies) != 4 || lock.LockfileVersion != 3 {
		t.Fatalf("%d dependencies, lockfileVersion %d", len(pkg.Dependencies), lock.LockfileVersion)
	}
	for name, version := range pkg.Dependencies {
		if strings.ContainsAny(version, "^~<>=*| ") || version == "" {
			t.Errorf("%s: %q is not an exact version", name, version)
		}
		got := lock.Packages["node_modules/"+name]
		if got.Version != version || !strings.HasPrefix(got.Integrity, "sha512-") || got.Resolved == "" {
			t.Errorf("%s: lock has %+v, want version %s with integrity and resolved", name, got, version)
		}
	}
}
