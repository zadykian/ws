package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	enginecli "github.com/zadykian/ws/engine/cli"
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
	confDir  = "../%s/mise/conf.d"
	// selfPath stands for the running ws, which a spec's self gives.
	selfPath = "/usr/local/bin/ws"
)

// roleFiles are the mise files a role may hold, each loaded by its env. A plain mise.toml would
// load in every env. mise.secrets.toml loads under the env secrets, which the secrets task adds.
// mise.self-contained.toml loads under the env self-contained, which a guest in the mode and every
// host load.
var roleFiles = []string{"mise.guest.toml", "mise.host.toml", "mise.secrets.toml", "mise.self-contained.toml"}

// specNames returns the names of the units that the roles' specs give the side for v, the side's
// values.
func specNames(side string, v any) ([]string, error) {
	e := kind.Env{M: effecttest.NewMachine(), U: &effecttest.Units{}, Out: &effecttest.Out{}, Data: v, Side: side, Self: selfPath}
	units, err := kind.Load(e, enginecli.SideRoles(os.DirFS(rolesDir), "mise."+side+".toml"))
	if err != nil {
		return nil, err
	}
	return kind.Names(units), nil
}

// specLines returns specNames for v, and fails the test where the specs do not load.
func specLines(t *testing.T, side string, v any) []string {
	names, err := specNames(side, v)
	if err != nil {
		t.Fatalf("the specs of the %s: %v", side, err)
	}
	return names
}

// sideValues returns the defaults of each side, by its name: what an empty values file gives.
func sideValues(t *testing.T) map[string]any {
	host, err := values.ParseHost(nil)
	if err != nil {
		t.Fatal(err)
	}
	return map[string]any{"guest": defaults(t), "host": host}
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
	for side, v := range sideValues(t) {
		file := "mise." + side + ".toml"
		lines := clitest.MiseLines(t, treeRoot+"/"+file)
		for name := range clitest.MiseLines(t, rolesDir+"/*/"+file) {
			if lines[name] {
				t.Errorf("\"ws:%s\" is in the root's %s and a role's", name, file)
			}
			lines[name] = true
		}
		clitest.CompareLines(t, "every "+side+" ("+file+")", specLines(t, side, v), lines)
	}

	// mise.toml loads on the Mac too, and mise.secrets.toml under the secrets task alone.
	for _, glob := range []string{treeRoot + "/mise.toml", rolesDir + "/*/mise.secrets.toml"} {
		for name := range clitest.MiseLines(t, glob) {
			t.Errorf("%s: \"ws:%s\" belongs in a side's mise file", glob, name)
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

// TestUnitsFollowNoValue holds the units of the specs to the same set on every machine of a side.
// mise's lines name each unit for every machine of the side. A unit that a value takes out of a spec would have
// a line on a machine where ws unit read refuses it. Such a unit stays, and its when turns it off.
// Each bool is negated, and each string emptied or set, one at a time.
func TestUnitsFollowNoValue(t *testing.T) {
	for side, base := range sideValues(t) {
		t.Run(side, func(t *testing.T) { unitsFollowNoValue(t, side, base) })
	}
}

// unitsFollowNoValue is TestUnitsFollowNoValue for one side, with the defaults of its values.
func unitsFollowNoValue(t *testing.T, side string, base any) {
	want := specLines(t, side, base)
	slices.Sort(want)
	for _, path := range turnable(reflect.TypeOf(base), nil) {
		v := reflect.New(reflect.TypeOf(base))
		v.Elem().Set(reflect.ValueOf(base))
		f := v.Elem().FieldByIndex(path)
		switch {
		case f.Kind() == reflect.Bool:
			f.SetBool(!f.Bool())
		case f.String() == "":
			f.SetString("/ws")
		default:
			f.SetString("")
		}
		got, err := specNames(side, v.Elem().Interface())
		if err != nil {
			// A value the specs refuse in this shape says nothing of which units they give.
			continue
		}
		slices.Sort(got)
		if !slices.Equal(got, want) {
			key := reflect.TypeOf(base).FieldByIndex(path).Tag.Get("yaml")
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

// TestTheSideLinksItsRoles holds each side's one conf.d link to the roles. mise loads a role's
// files from a directory of conf.d alone, and a file there in every env. A link elsewhere
// would drop every role's lines with no error.
func TestTheSideLinksItsRoles(t *testing.T) {
	for _, side := range []string{"guest", "host"} {
		clitest.LinksRoles(t, fmt.Sprintf(confDir, side), "../../roles")
	}
	if dirs := roleDirs(t); !slices.Contains(dirs, "frame") {
		t.Errorf("the server's roles %v lack the frame", dirs)
	}
}
