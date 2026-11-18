package cli

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"

	"github.com/zadykian/ws/engine/cli/clitest"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/roles/client/macos/facts"
	"github.com/zadykian/ws/roles/client/macos/values"
)

// The paths below are from this package's directory: the roles are its parent, unitsDir in the
// checkout, and the tree's root is four levels up.
const (
	macRolesDir = ".."
	macTreeRoot = "../../../.."
	macConfDir  = macTreeRoot + "/mise/conf.d"
	// macRoleFile is a role's one mise file: the env mac loads it, and no env of the server's.
	macRoleFile = "mise.mac.toml"
)

// macFacts are the facts of a Mac with the servers given and Ghostty's default config.
func macFacts(servers ...facts.Server) facts.Mac {
	return facts.Mac{User: "v", Home: "/Users/v", Servers: servers, Ghostty: facts.Ghostty{
		Config: "/Users/v/Library/Application Support/com.mitchellh.ghostty/config.ghostty"}}
}

// twoServerFacts are the facts of a Mac with two servers, so that a spec's loop over them shows.
func twoServerFacts() facts.Mac {
	return macFacts(facts.Server{Name: "lab", Conf: values.Conf{Slot: "1", PublicKey: "lab-key"}},
		facts.Server{Name: "ws", Conf: values.Conf{Slot: "0", PublicKey: "ws-key"}})
}

// macSpecNames returns the names of the units that the Mac's specs give for f.
func macSpecNames(t *testing.T, f facts.Mac) []string {
	e := kind.Env{M: effecttest.NewMachine(), Out: &effecttest.Out{}, Data: f, System: kind.MacOS,
		Self: "/Users/v/.local/share/mise/installs/http-ws/0.12.0/ws", User: f.User}
	units, err := kind.Load(e, os.DirFS(macRolesDir))
	if err != nil {
		t.Fatalf("the Mac's specs: %v", err)
	}
	return kind.Names(units)
}

// TestMacSpecsMatchMiseLines holds the Mac's lines and its specs together, as ws unit read
// --whole checks on a Mac. A Mac with no server and one with two give the same units, as the
// lines do not follow the servers: a unit's spec loops over them within it.
func TestMacSpecsMatchMiseLines(t *testing.T) {
	want := macSpecNames(t, twoServerFacts())
	if alone := macSpecNames(t, macFacts()); !slices.Equal(alone, want) {
		t.Errorf("the units of a Mac with no server, %v, differ from those of two, %v", alone, want)
	}
	lines := clitest.MiseLines(t, macTreeRoot+"/"+macRoleFile)
	for name := range clitest.MiseLines(t, macRolesDir+"/*/"+macRoleFile) {
		if lines[name] {
			t.Errorf("\"ws:%s\" is in the root's %s and a role's", name, macRoleFile)
		}
		lines[name] = true
	}
	clitest.CompareLines(t, "the Mac (mise.mac.toml)", want, lines)
}

// macRoleDirs returns the directories of the Mac's roles that hold a spec or a mise file, and
// fails the test on a mise file other than mise.mac.toml.
func macRoleDirs(t *testing.T) []string {
	entries, err := os.ReadDir(macRolesDir)
	if err != nil {
		t.Fatal(err)
	}
	var dirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(macRolesDir, e.Name())
		files, err := filepath.Glob(filepath.Join(dir, "mise*.toml"))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			if filepath.Base(f) != macRoleFile {
				t.Errorf("%s: a role of the Mac's holds %s alone", f, macRoleFile)
			}
		}
		if _, err := os.Stat(filepath.Join(dir, "units.toml")); err == nil || len(files) > 0 {
			dirs = append(dirs, e.Name())
		}
	}
	return dirs
}

// TestEachMacRoleIsLinked holds mise/conf.d and the Mac's roles together. mise loads a role's files
// through its link alone: a role without one would drop its lines with no error.
func TestEachMacRoleIsLinked(t *testing.T) {
	dirs := macRoleDirs(t)
	if !slices.Contains(dirs, "frame") {
		t.Errorf("the Mac's roles %v lack the frame", dirs)
	}
	for _, dir := range dirs {
		link := filepath.Join(macConfDir, "mac-"+dir)
		target, err := os.Readlink(link)
		if want := "../../roles/client/macos/" + dir; err != nil || target != want {
			t.Errorf("%s leads to %q (%v), want a link to %s", link, target, err, want)
		}
	}
	entries, err := os.ReadDir(macConfDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
	for _, e := range entries {
		dir, ok := strings.CutPrefix(e.Name(), "mac-")
		if ok && !slices.Contains(dirs, dir) {
			t.Errorf("%s: no role %s of the Mac's holds a spec or a mise file", filepath.Join(macConfDir, e.Name()), dir)
		}
	}
}

// TestMacConfigHoldsNoRootPath keeps mise's own files and dotfiles out of the Mac's config. mise
// reads its files of root's through sudo on every run, which would ask a password of a run that
// changes nothing. Its dotfiles replace the user's own file with no guard. ws's units hold both.
func TestMacConfigHoldsNoRootPath(t *testing.T) {
	files, err := filepath.Glob(macRolesDir + "/*/" + macRoleFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range append(files, macTreeRoot+"/"+macRoleFile) {
		text, err := os.ReadFile(f) //nolint:gosec // a file of the tree, found by the test
		if err != nil {
			t.Fatal(err)
		}
		var c struct {
			Bootstrap map[string]any `toml:"bootstrap"`
			Dotfiles  map[string]any `toml:"dotfiles"`
		}
		if err := toml.Unmarshal(text, &c); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for _, part := range []string{"files", "directories"} {
			if _, ok := c.Bootstrap[part]; ok {
				t.Errorf("%s: [bootstrap.%s] is mise's, read through sudo: make it a unit of ws's", f, part)
			}
		}
		if len(c.Dotfiles) > 0 {
			t.Errorf("%s: [dotfiles] replaces the user's own file with no guard: make it a unit of ws's", f)
		}
	}
}
