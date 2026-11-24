package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/cli/clitest"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/ubuntu/values"
)

// The paths below are from this package's directory. The roles are its sibling roles, unitsDir in
// the checkout. The guest's conf.d is in its sibling guest, and the tree's root is two levels up.
const (
	rolesDir = "../roles"
	treeRoot = "../.."
	confDir  = "../guest/mise/conf.d"
	// selfPath stands for the running ws, which a spec's self gives.
	selfPath = "/usr/local/bin/ws"
)

// roleFiles are the mise files a role may hold, each loaded by its env. A plain mise.toml would
// load in every env. mise.secrets.toml loads under the env secrets, which the secrets task adds.
var roleFiles = []string{"mise.guest.toml", "mise.secrets.toml"}

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
		t.Fatalf("the specs: %v", err)
	}
	return names
}

// defaults returns the values of an empty file, which fills the keys that derive gives.
func defaults(t *testing.T) values.Values {
	v, err := values.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// TestSpecsMatchMiseLines holds mise's lines and the specs together, so that mise plans each unit
// ws knows, as ws unit verify checks on a machine. The server's files hold each unit of the specs.
func TestSpecsMatchMiseLines(t *testing.T) {
	every := specLines(t, defaults(t))

	server := clitest.MiseLines(t, treeRoot+"/mise.guest.toml")
	for name := range clitest.MiseLines(t, rolesDir+"/*/mise.guest.toml") {
		if server[name] {
			t.Errorf("\"ws:%s\" is in the root's mise.guest.toml and a role's", name)
		}
		server[name] = true
	}
	clitest.CompareLines(t, "every server (mise.guest.toml)", every, server)

	// mise.toml loads on the Mac too, and mise.secrets.toml under the secrets task alone.
	for _, glob := range []string{treeRoot + "/mise.toml", rolesDir + "/*/mise.secrets.toml"} {
		for name := range clitest.MiseLines(t, glob) {
			t.Errorf("%s: \"ws:%s\" belongs in a mise.guest.toml", glob, name)
		}
	}
}

// turnable returns the path of each bool and string field below Values.
func turnable(t reflect.Type, at []int) [][]int {
	var paths [][]int
	for i := range t.NumField() {
		f := t.Field(i)
		path := append(slices.Clone(at), i)
		switch {
		case !f.IsExported():
		case f.Type.Kind() == reflect.Struct:
			paths = append(paths, turnable(f.Type, path)...)
		case f.Type.Kind() == reflect.Bool, f.Type.Kind() == reflect.String:
			paths = append(paths, path)
		}
	}
	return paths
}

// TestUnitsFollowNoValue holds the units of the specs to the same set on every machine.
// mise's lines name each unit for every server. A unit that a value takes out of a spec would have
// a line on a machine where ws unit read refuses it. Such a unit stays, and its when turns it off.
// Each bool is negated, and each string emptied or set, one at a time.
func TestUnitsFollowNoValue(t *testing.T) {
	base := defaults(t)
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
			t.Errorf("%s turned over gives the units %v, not %v", key, got, want)
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
		if _, err := os.Stat(filepath.Join(dir, "units.toml.tmpl")); err == nil || len(files) > 0 {
			dirs = append(dirs, e.Name())
		}
	}
	return dirs
}

// TestTheSideLinksItsRoles holds the server's one conf.d link to its roles. mise loads a role's
// files from a directory of conf.d alone, and a file there in every env. A link elsewhere
// would drop every role's lines with no error.
func TestTheSideLinksItsRoles(t *testing.T) {
	clitest.LinksRoles(t, confDir, "../../roles")
	if dirs := roleDirs(t); !slices.Contains(dirs, "frame") {
		t.Errorf("the server's roles %v lack the frame", dirs)
	}
}
