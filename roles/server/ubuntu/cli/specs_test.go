package cli

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// The paths below are from this package's directory: the roles are its parent, unitsDir in the
// checkout, and the tree's root is four levels up.
const (
	rolesDir = ".."
	treeRoot = "../../../.."
	confDir  = treeRoot + "/mise/conf.d"
	// selfPath stands for the running ws, which a spec's self gives.
	selfPath = "/usr/local/bin/ws"
)

// roleFiles are the mise files a role may hold, each loaded by its env. A plain mise.toml would
// load on the Mac too.
var roleFiles = []string{"mise.server.toml", "mise.booted.toml", "mise.account.toml"}

// specNames returns the names of the units that the roles' specs give for v.
func specNames(v values.Values) ([]string, error) {
	e := kind.Env{M: effecttest.NewMachine(), U: &effecttest.Units{}, Out: &effecttest.Out{}, Data: v, Self: selfPath}
	units, err := kind.Load(e, os.DirFS(rolesDir))
	if err != nil {
		return nil, err
	}
	return kind.Names(units), nil
}

// specLines returns specNames for v, and fails the test where the specs do not load.
func specLines(t *testing.T, v values.Values) []string {
	names, err := specNames(v)
	if err != nil {
		t.Fatalf("the specs for %s: %v", v.User, err)
	}
	return names
}

// rootAndAccount returns the values of a machine whose WS_USER is root, and the defaults, whose
// WS_USER is an account.
func rootAndAccount(t *testing.T) (root, account values.Values) {
	root, err := values.Parse([]byte("WS_USER: root\n"))
	if err != nil {
		t.Fatal(err)
	}
	account, err = values.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	return root, account
}

// miseLines returns the units that the files of glob name under [bootstrap.packages], each
// "ws:KIND/NAME" without its prefix. A unit in two of the files fails the test.
func miseLines(t *testing.T, glob string) map[string]bool {
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
				Packages map[string]string `toml:"packages"`
			} `toml:"bootstrap"`
		}
		if err := toml.Unmarshal(text, &c); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for pkg := range c.Bootstrap.Packages {
			name, ok := strings.CutPrefix(pkg, "ws:")
			if !ok {
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

// compareLines reports each unit of want that got lacks, and each of got that want lacks, as the
// line to add to or drop from the files of where.
func compareLines(t *testing.T, where string, want []string, got map[string]bool) {
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

// TestSpecsMatchMiseLines holds mise's lines and the specs together, so that mise plans each unit
// ws knows, as ws unit verify checks on a machine. The server's files hold each unit for every
// server. The account's files hold each unit that the spec gives where WS_USER is not root, as the
// miserc loads them there alone. A unit for root alone has no env. The server's files load on an
// account's machine too, where ws unit read refuses a unit that no spec gives.
func TestSpecsMatchMiseLines(t *testing.T) {
	root, account := rootAndAccount(t)
	every := specLines(t, root)
	accountLines := specLines(t, account)
	for _, name := range every {
		if !slices.Contains(accountLines, name) {
			t.Errorf("%s: the specs give it for root alone, and no env loads a unit for root alone", name)
		}
	}
	accountOnly := []string{}
	for _, name := range accountLines {
		if !slices.Contains(every, name) {
			accountOnly = append(accountOnly, name)
		}
	}

	server := miseLines(t, treeRoot+"/mise.server.toml")
	for name := range miseLines(t, rolesDir+"/*/mise.server.toml") {
		if server[name] {
			t.Errorf("\"ws:%s\" is in the root's mise.server.toml and a role's", name)
		}
		server[name] = true
	}
	compareLines(t, "every server (mise.server.toml)", every, server)
	compareLines(t, "an account (mise.account.toml)", accountOnly, miseLines(t, rolesDir+"/*/mise.account.toml"))

	// A unit skips itself without booted systemd, and mise.toml loads on the Mac too.
	for _, glob := range []string{treeRoot + "/mise.toml", rolesDir + "/*/mise.booted.toml"} {
		for name := range miseLines(t, glob) {
			t.Errorf("%s: \"ws:%s\" belongs in a mise.server.toml or mise.account.toml", glob, name)
		}
	}
}

// turnable returns the path of each bool and string field below Values, but those of Shared: the
// keys that one machine sets and another leaves. WS_USER picks the env instead.
func turnable(t reflect.Type, at []int) [][]int {
	var paths [][]int
	for i := range t.NumField() {
		f := t.Field(i)
		path := append(slices.Clone(at), i)
		switch {
		case !f.IsExported() || f.Type == reflect.TypeFor[values.Shared]():
		case f.Type.Kind() == reflect.Struct:
			paths = append(paths, turnable(f.Type, path)...)
		case f.Type.Kind() == reflect.Bool, f.Type.Kind() == reflect.String:
			paths = append(paths, path)
		}
	}
	return paths
}

// TestUnitsFollowTheUserAlone holds the units of the specs to WS_USER alone. mise's lines go by
// env, and the envs follow WS_USER alone. A unit that another value takes out of a spec would have
// a line on a machine where ws unit read refuses it. Such a unit stays, and its when turns it off.
// Each bool is negated, and each string emptied or set, one at a time.
func TestUnitsFollowTheUserAlone(t *testing.T) {
	root, account := rootAndAccount(t)
	for _, base := range []values.Values{root, account} {
		want := specLines(t, base)
		slices.Sort(want)
		for _, path := range turnable(reflect.TypeFor[values.Values](), nil) {
			v := base
			f := reflect.ValueOf(&v).Elem().FieldByIndex(path)
			switch {
			case f.Kind() == reflect.Bool:
				f.SetBool(!f.Bool())
			case f.String() == "":
				f.SetString("/ws")
			default:
				f.SetString("")
			}
			got, err := specNames(v)
			if err != nil {
				// A value the specs refuse in this shape says nothing of which units they give.
				continue
			}
			slices.Sort(got)
			if !slices.Equal(got, want) {
				key := reflect.TypeFor[values.Values]().FieldByIndex(path).Tag.Get("yaml")
				t.Errorf("for %s, %s turned over gives the units %v, not %v", base.User, key, got, want)
			}
		}
	}
}

// roleDirs returns the directories of the roles that hold a spec or a mise file, and fails the test
// on a mise file that no env of the server loads.
func roleDirs(t *testing.T) []string {
	entries, err := os.ReadDir(rolesDir)
	if err != nil {
		t.Fatal(err)
	}
	var dirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(rolesDir, e.Name())
		files, err := filepath.Glob(filepath.Join(dir, "mise*.toml"))
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			if !slices.Contains(roleFiles, filepath.Base(f)) {
				t.Errorf("%s: a role's mise files are %s", f, strings.Join(roleFiles, ", "))
			}
		}
		if _, err := os.Stat(filepath.Join(dir, "units.toml")); err == nil || len(files) > 0 {
			dirs = append(dirs, e.Name())
		}
	}
	return dirs
}

// TestEachRoleIsLinked holds mise/conf.d and the roles together. mise loads a role's files through
// its link alone: a role without one would drop its lines with no error.
func TestEachRoleIsLinked(t *testing.T) {
	dirs := roleDirs(t)
	for _, dir := range dirs {
		link := filepath.Join(confDir, "server-"+dir)
		target, err := os.Readlink(link)
		if want := "../../roles/server/ubuntu/" + dir; err != nil || target != want {
			t.Errorf("%s leads to %q (%v), want a link to %s", link, target, err, want)
		}
	}
	entries, err := os.ReadDir(confDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
	for _, e := range entries {
		dir, ok := strings.CutPrefix(e.Name(), "server-")
		if ok && !slices.Contains(dirs, dir) {
			t.Errorf("%s: no role %s holds a spec or a mise file", filepath.Join(confDir, e.Name()), dir)
		}
	}
}
