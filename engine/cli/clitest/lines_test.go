package clitest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// recorder keeps the errors that a check reports, so that a test can expect them; the rest of
// testing.TB is the running test's.
type recorder struct {
	testing.TB
	errs []string
}

func (r *recorder) Errorf(format string, args ...any) {
	r.errs = append(r.errs, fmt.Sprintf(format, args...))
}

// TestMiseLinesKeepsTheUnitsBesideAPackageTable holds the units of a role file that also pins an
// app, whose package is a table where a unit's is a string.
func TestMiseLinesKeepsTheUnitsBesideAPackageTable(t *testing.T) {
	dir := t.TempDir()
	text := "[bootstrap.packages]\n\"ws:state/a\" = \"in-sync\"\n\"brew-cask:x\" = \"latest\"\n" +
		"[bootstrap.packages.\"macos-app:x\"]\nversion = \"1\"\nurl = \"https://example.com/x.zip\"\n" +
		"sha256 = \"" + strings.Repeat("0", 64) + "\"\nartifact = \"X.app\"\nadopt = true\n"
	if err := os.WriteFile(filepath.Join(dir, "mise.macos.toml"), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	lines := MiseLines(t, filepath.Join(dir, "*.toml"))
	if len(lines) != 1 || !lines["state/a"] {
		t.Errorf("the units of the file are %v, want state/a alone", lines)
	}
}

// TestLinksRolesNamesAFileOrAnotherTarget holds a side's link to its directory of roles, with
// directories alone in it.
func TestLinksRolesNamesAFileOrAnotherTarget(t *testing.T) {
	side := t.TempDir()
	if err := os.MkdirAll(filepath.Join(side, "roles", "frame"), 0o750); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(side, "conf.d")
	if err := os.Symlink("roles", link); err != nil {
		t.Fatal(err)
	}
	r := &recorder{TB: t}
	if LinksRoles(r, link, "roles"); len(r.errs) != 0 {
		t.Errorf("a link to roles of directories alone: %v", r.errs)
	}
	if LinksRoles(r, link, "other"); len(r.errs) != 1 {
		t.Errorf("a link elsewhere gives %v, want one error", r.errs)
	}
	if err := os.WriteFile(filepath.Join(side, "roles", "mise.toml"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	r = &recorder{TB: t}
	if LinksRoles(r, link, "roles"); len(r.errs) != 1 {
		t.Errorf("a file beside the roles gives %v, want one error", r.errs)
	}
}

// TestFrameKeysNamesASharedKey fails a role's entry of a path that the frame declares, and passes
// two apart and a shared table that merges.
func TestFrameKeysNamesASharedKey(t *testing.T) {
	dir := t.TempDir()
	write := func(name, text string) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	packages := "[bootstrap.packages]\n\"ws:state/a\" = \"in-sync\"\n"
	frame := write("frame.toml", packages+"[bootstrap.files.\"/etc/x\"]\ncontent = \"x\"\n")
	write("apart.role", packages+"[bootstrap.files.\"/etc/y\"]\ncontent = \"y\"\n")
	r := &recorder{TB: t}
	if FrameKeys(r, []string{frame}, filepath.Join(dir, "*.role")); len(r.errs) != 0 {
		t.Errorf("a role apart from the frame: %v", r.errs)
	}
	write("shared.role", "[bootstrap.files.\"/etc/x\"]\ncontent = \"z\"\n")
	if FrameKeys(r, []string{frame}, filepath.Join(dir, "*.role")); len(r.errs) != 1 {
		t.Errorf("a role that shares /etc/x gives %v, want one error", r.errs)
	}
}
