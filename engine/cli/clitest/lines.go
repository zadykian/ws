package clitest

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// MiseLines returns the units that the files of glob name under [bootstrap.packages], each
// "ws:KIND/NAME" without its prefix. A package whose value is a table, as mise's macos-app, is no
// unit. A unit in two of the files fails the test.
func MiseLines(t testing.TB, glob string) map[string]bool {
	files, err := filepath.Glob(glob)
	if err != nil {
		t.Fatal(err)
	}
	lines := map[string]bool{}
	where := map[string]string{}
	for _, f := range files {
		text, err := os.ReadFile(f) //nolint:gosec // a file of the tree, found by the test
		if err != nil {
			t.Fatal(err)
		}
		var c struct {
			Bootstrap struct {
				Packages map[string]any `toml:"packages"`
			} `toml:"bootstrap"`
		}
		if err := toml.Unmarshal(text, &c); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for pkg, value := range c.Bootstrap.Packages {
			name, ok := strings.CutPrefix(pkg, "ws:")
			if _, line := value.(string); !ok || !line {
				continue
			}
			if other, twice := where[name]; twice {
				t.Errorf("%q is in %s and %s: one role names a unit", pkg, other, f)
			}
			lines[name], where[name] = true, f
		}
	}
	return lines
}

// CompareLines reports each unit of want that got lacks, and each of got that want lacks, as the
// line to add to or drop from the files of where.
func CompareLines(t testing.TB, where string, want []string, got map[string]bool) {
	for _, name := range want {
		if !got[name] {
			t.Errorf("%s: missing \"ws:%s\" = %q", where, name, "in-sync")
		}
	}
	extra := []string{}
	for name := range got {
		if !slices.Contains(want, name) {
			extra = append(extra, name)
		}
	}
	slices.Sort(extra)
	for _, name := range extra {
		t.Errorf("%s: extra \"ws:%s\" = %q, which no spec gives there", where, name, "in-sync")
	}
}

// LinksRoles fails the test where link, a side's conf.d, does not read want, or where its
// directory holds a file. mise loads each directory there as a role's, and a file beside them in
// every env.
func LinksRoles(t testing.TB, link, want string) {
	target, err := os.Readlink(link)
	if err != nil || target != want {
		t.Errorf("%s leads to %q (%v), want a link to %s", link, target, err, want)
		return
	}
	entries, err := os.ReadDir(link)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			t.Errorf("%s: mise would load it in every env: a role is a directory", filepath.Join(link, e.Name()))
		}
	}
}

// FrameKeys fails the test on each key that a file of glob and a file of frames both set. A key is
// a setting, a tool, or an entry of a [bootstrap] table. The frames load before the roles, so a
// role's key would win over the frame's. [bootstrap.packages] merges across files and is left out.
func FrameKeys(t testing.TB, frames []string, glob string) {
	framed := map[string]string{}
	for _, f := range frames {
		for _, key := range configKeys(t, f) {
			framed[key] = f
		}
	}
	files, err := filepath.Glob(glob)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		for _, key := range configKeys(t, f) {
			if frame, ok := framed[key]; ok {
				t.Errorf("%s sets %s, which %s sets: the role's would win over the frame's", f, key, frame)
			}
		}
	}
}

// configKeys returns the keys of mise's file f that a later file would replace whole, each as
// TABLE.KEY.
func configKeys(t testing.TB, f string) []string {
	text, err := os.ReadFile(f) //nolint:gosec // a file of the tree, found by the test
	if err != nil {
		t.Fatal(err)
	}
	var c struct {
		Settings  map[string]any `toml:"settings"`
		Tools     map[string]any `toml:"tools"`
		Bootstrap map[string]any `toml:"bootstrap"`
	}
	if err := toml.Unmarshal(text, &c); err != nil {
		t.Fatalf("%s: %v", f, err)
	}
	var keys []string
	for key := range c.Settings {
		keys = append(keys, "settings."+key)
	}
	for key := range c.Tools {
		keys = append(keys, "tools."+key)
	}
	for part, table := range c.Bootstrap {
		entries, ok := table.(map[string]any)
		if part == "packages" || !ok {
			continue
		}
		for key := range entries {
			keys = append(keys, "bootstrap."+part+"."+key)
		}
	}
	return keys
}
