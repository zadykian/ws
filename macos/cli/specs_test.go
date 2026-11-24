package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"

	"github.com/zadykian/ws/engine/cli/clitest"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/macos/facts"
	"github.com/zadykian/ws/macos/values"
)

// The paths below are from this package's directory. The roles are its sibling macos, unitsDir in
// the checkout. The side's conf.d and lock are its sibling mise's, and the tree's root is three
// levels up.
const (
	macRolesDir = "../roles"
	macTreeRoot = "../.."
	macConfDir  = "../mise/conf.d"
	macLock     = "../mise/mise.macos.lock"
	// macRoleFile is a role's mise file: the env macos loads it, and no env of the server's.
	macRoleFile = "mise.macos.toml"
	// appsFile is the one other file a role holds: the env apps loads it, after the bootstrap's own run.
	appsFile = "mise.apps.toml"
)

// macMiseFiles returns the mise files of the Mac's roles, of the env macos and of the env apps.
func macMiseFiles(t *testing.T) []string {
	var files []string
	for _, name := range []string{macRoleFile, appsFile} {
		found, err := filepath.Glob(macRolesDir + "/*/" + name)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, found...)
	}
	return files
}

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
	if !lines["state/ws-binary"] {
		t.Errorf("the root's %s lacks \"ws:state/ws-binary\": a run from /opt/ws loads no role, and "+
			"calls the hook that fails it for a line of the root's alone", macRoleFile)
	}
	for name := range clitest.MiseLines(t, macRolesDir+"/*/"+macRoleFile) {
		if lines[name] {
			t.Errorf("\"ws:%s\" is in the root's %s and a role's", name, macRoleFile)
		}
		lines[name] = true
	}
	clitest.CompareLines(t, "the Mac (mise.macos.toml)", want, lines)
}

// TestTheAppsEnvHoldsNoUnit keeps ws's units out of the env apps' files. The units run in the first
// bootstrap, and a unit's line that only the second loads would fail the run that lacks it.
func TestTheAppsEnvHoldsNoUnit(t *testing.T) {
	if lines := clitest.MiseLines(t, macRolesDir+"/*/"+appsFile); len(lines) > 0 {
		t.Errorf("a file %s holds the units' lines %v: they belong in %s", appsFile, lines, macRoleFile)
	}
}

// macRoleDirs returns the directories of the Mac's roles that hold a spec or a mise file, and
// fails the test on any mise file but those two.
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
			if base := filepath.Base(f); base != macRoleFile && base != appsFile {
				t.Errorf("%s: a role of the Mac's holds %s and %s alone", f, macRoleFile, appsFile)
			}
		}
		if _, err := os.Stat(filepath.Join(dir, "units.toml.tmpl")); err == nil || len(files) > 0 {
			dirs = append(dirs, e.Name())
		}
	}
	return dirs
}

// TestTheMacLinksItsRoles holds the Mac's one conf.d link to its roles. mise loads a role's files
// from a directory of conf.d alone, and a file there in every env. A link elsewhere would drop
// every role's lines with no error.
func TestTheMacLinksItsRoles(t *testing.T) {
	clitest.LinksRoles(t, macConfDir, "../roles")
	if dirs := macRoleDirs(t); !slices.Contains(dirs, "frame") {
		t.Errorf("the Mac's roles %v lack the frame", dirs)
	}
}

// TestNoMacRoleSetsAFramesKey keeps each of the Mac's roles off the keys of its frame, which loads
// before the roles through the side's link.
func TestNoMacRoleSetsAFramesKey(t *testing.T) {
	for _, name := range []string{macRoleFile, appsFile} {
		clitest.FrameKeys(t, []string{macTreeRoot + "/mise.toml", macTreeRoot + "/" + macRoleFile}, macRolesDir+"/*/"+name)
	}
}

// TestMacConfigHoldsNoRootPath keeps mise's own root paths and dotfiles out of the Mac's config.
// mise reads its files of root's through sudo on every run, which would ask a password of a run
// that changes nothing. Its dotfiles replace the user's own file with no guard. ws's units hold
// both. A file under ~/ with no owner or group is the user's: mise reads and writes it as the user.
func TestMacConfigHoldsNoRootPath(t *testing.T) {
	for _, f := range append(macMiseFiles(t), macTreeRoot+"/"+macRoleFile) {
		text, err := os.ReadFile(f) //nolint:gosec // a file of the tree, found by the test
		if err != nil {
			t.Fatal(err)
		}
		var c struct {
			Bootstrap struct {
				Files       map[string]map[string]any `toml:"files"`
				Directories map[string]any            `toml:"directories"`
			} `toml:"bootstrap"`
			Dotfiles map[string]any `toml:"dotfiles"`
		}
		if err := toml.Unmarshal(text, &c); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for target, part := range c.Bootstrap.Files {
			_, owner := part["owner"]
			_, group := part["group"]
			if !strings.HasPrefix(target, "~/") || owner || group {
				t.Errorf("%s: [bootstrap.files.%q] is root's, read through sudo: make it a unit of ws's, or a path under ~/ with no owner or group", f, target)
			}
		}
		if len(c.Bootstrap.Directories) > 0 {
			t.Errorf("%s: [bootstrap.directories] is mise's, and mise makes no missing parent: make it a unit of ws's", f)
		}
		if len(c.Dotfiles) > 0 {
			t.Errorf("%s: [dotfiles] replaces the user's own file with no guard: make it a unit of ws's", f)
		}
	}
}

// TestMacLockHoldsEachTool keeps macos/mise/mise.macos.lock beside the Mac's tools. The
// bootstrap runs with the tree's setting locked. That refuses a tool that the lock lacks, and
// installs one by its url and checksum alone, so that no tool goes in unchecked. An entry that no role pins, as
// after a change of a tool's backend, is stale.
func TestMacLockHoldsEachTool(t *testing.T) {
	text, err := os.ReadFile(macLock)
	if err != nil {
		t.Fatal(err)
	}
	var lock struct {
		Tools map[string][]map[string]any `toml:"tools"`
	}
	if err := toml.Unmarshal(text, &lock); err != nil {
		t.Fatalf("mise.macos.lock: %v", err)
	}
	pinned := map[string]bool{}
	for _, f := range macMiseFiles(t) {
		text, err := os.ReadFile(f) //nolint:gosec // a file of the tree, found by the test
		if err != nil {
			t.Fatal(err)
		}
		var c struct {
			Tools map[string]string `toml:"tools"`
		}
		if err := toml.Unmarshal(text, &c); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for name, version := range c.Tools {
			pinned[name] = true
			i := slices.IndexFunc(lock.Tools[name], func(e map[string]any) bool { return e["version"] == version })
			if i < 0 {
				t.Errorf("%s: %s %s is not in macos/mise/mise.macos.lock: run mise lock for macos-arm64", f, name, version)
				continue
			}
			mac, _ := lock.Tools[name][i]["platforms.macos-arm64"].(map[string]any)
			url, _ := mac["url"].(string)
			sum, _ := mac["checksum"].(string)
			if !strings.HasPrefix(url, "https://") || !strings.HasPrefix(sum, "sha256:") {
				t.Errorf("%s: the lock's %s %s lacks a url and a sha256 for macos-arm64: %v", f, name, version, mac)
			}
		}
	}
	if len(pinned) == 0 {
		t.Error("no role of the Mac's names a tool, want gh at least")
	}
	for name := range lock.Tools {
		if !pinned[name] {
			t.Errorf("macos/mise/mise.macos.lock holds %s, which no role pins: run mise lock for macos-arm64", name)
		}
	}
}
