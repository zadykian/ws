package kind

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

// values stands for a side's values, which fill the specs.
type values struct{ User, Dir string }

// testEnv is an Env on a fresh machine, with the values of user u.
func testEnv() (Env, *effecttest.Machine, *effecttest.Units, *effecttest.Out) {
	m, u, out := effecttest.NewMachine(), &effecttest.Units{}, &effecttest.Out{}
	return Env{M: m, U: u, Out: out, Data: values{User: "u", Dir: "/srv"}}, m, u, out
}

// tree is a root of specs: each role's spec, ROLE/units.toml.tmpl, with the files beside it.
func tree(files map[string]string) fstest.MapFS {
	fsys := fstest.MapFS{}
	for name, text := range files {
		fsys[name] = &fstest.MapFile{Data: []byte(text)}
	}
	return fsys
}

// loadOne loads the spec of one role, r, with its files, and fails the test on an error.
func loadOne(t *testing.T, e Env, spec string, files map[string]string) []Unit {
	all := map[string]string{"r/units.toml.tmpl": spec}
	for name, text := range files {
		all["r/"+name] = text
	}
	units, err := Load(e, tree(all))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return units
}

// headlines plans the step and returns its changes' headlines.
func headlines(t *testing.T, s step.Step) []string {
	p, err := s.Plan(t.Context())
	if err != nil {
		t.Fatalf("plan %s: %v", s.Name(), err)
	}
	var got []string
	for _, c := range p.Changes {
		got = append(got, step.Headline(c))
	}
	return got
}

func TestLoadReadsEachRolesSpec(t *testing.T) {
	e, _, _, _ := testEnv()
	units, err := Load(e, tree(map[string]string{
		"a/units.toml.tmpl":     "[file.one]\npath = \"/x/1\"\nsource = \"f\"\n",
		"a/f":                   "1",
		"b/units.toml.tmpl":     "[file.two]\npath = \"/x/2\"\nsource = \"f\"\n",
		"b/f":                   "2",
		"units.toml.tmpl":       "[file.root]\npath = \"/x/3\"\nsource = \"f\"\n",
		"b/c/units.toml.tmpl":   "[file.deep]\npath = \"/x/4\"\nsource = \"f\"\n",
		"a/other.toml":          "[file.other]\npath = \"/x/5\"\nsource = \"f\"\n",
		"b/units.toml.tmpl.bak": "",
	}))
	if got := Names(units); err != nil || !slices.Equal(got, []string{"file/one", "file/two"}) {
		t.Errorf("load = %q, %v; want file/one and file/two alone", got, err)
	}
}

func TestLoadRefusesAnUnknownField(t *testing.T) {
	e, _, _, _ := testEnv()
	_, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": "[file.a]\npth = \"/x\"\n"}))
	if err == nil || !strings.Contains(err.Error(), "file.a.pth") || !strings.Contains(err.Error(), "r/units.toml.tmpl") {
		t.Errorf("load = %v, want an error that names r/units.toml.tmpl and file.a.pth", err)
	}
	_, err = Load(e, tree(map[string]string{"r/units.toml.tmpl": "[nokind.a]\npath = \"/x\"\n"}))
	if err == nil || !strings.Contains(err.Error(), "nokind") {
		t.Errorf("load = %v, want an error that names the kind nokind", err)
	}
	_, err = Load(e, tree(map[string]string{"r/units.toml.tmpl": "[file.a]\npath = \"/x\"\nmode = \"rw\"\n"}))
	if err == nil || !strings.Contains(err.Error(), "file.a.mode (line 3") {
		t.Errorf("load = %v, want an error that names file.a.mode and its line", err)
	}
}

func TestLoadRefusesANameInTwoSpecs(t *testing.T) {
	e, _, _, _ := testEnv()
	spec := "[file.a]\npath = \"/x\"\nsource = \"f\"\n"
	_, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": spec, "r/f": "", "s/units.toml.tmpl": spec, "s/f": ""}))
	if err == nil || !strings.Contains(err.Error(), "file/a is in r/units.toml.tmpl too") {
		t.Errorf("load = %v, want the unit named in both specs", err)
	}
}

func TestLoadRefusesANameThatIsNoPlainWord(t *testing.T) {
	e, _, _, _ := testEnv()
	for _, name := range []string{`"a/b"`, `""`, `"a b"`, `"-a"`} {
		_, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": "[file." + name + "]\npath = \"/x\"\nsource = \"f\"\n", "r/f": ""}))
		if err == nil || !strings.Contains(err.Error(), "a name takes") {
			t.Errorf("load of %s = %v, want the name refused", name, err)
		}
	}
}

func TestOrderMovesAUnitAfterItsAfter(t *testing.T) {
	e, _, _, _ := testEnv()
	spec := ""
	for _, u := range []struct{ name, after string }{{"a", "file/c"}, {"b", ""}, {"c", ""}, {"d", "file/a"}, {"e", "file/c"}, {"f", ""}} {
		spec += "[file." + u.name + "]\npath = \"/x/" + u.name + "\"\nsource = \"f\"\nafter = \"" + u.after + "\"\n"
	}
	units := loadOne(t, e, spec, map[string]string{"f": ""})
	if got, want := Names(units), []string{"file/b", "file/c", "file/a", "file/d", "file/e", "file/f"}; !slices.Equal(got, want) {
		t.Errorf("order = %q, want %q", got, want)
	}
	if got := units[2].Step.WaitsOn(); got != "file/c" {
		t.Errorf("file/a waits on %q, want file/c", got)
	}
}

func TestOrderRefusesAnUnknownAfterAndALoop(t *testing.T) {
	e, _, _, _ := testEnv()
	for spec, want := range map[string]string{
		"[file.a]\npath = \"/a\"\nsource = \"f\"\nafter = \"file/z\"\n": "file/a waits on file/z, which no spec holds",
		"[file.a]\npath = \"/a\"\nsource = \"f\"\nafter = \"file/b\"\n" +
			"[file.b]\npath = \"/b\"\nsource = \"f\"\nafter = \"file/a\"\n" +
			"[file.c]\npath = \"/c\"\nsource = \"f\"\n": "loop: file/a (after file/b), file/b (after file/a)",
	} {
		if _, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": spec, "r/f": ""})); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("load = %v, want %q", err, want)
		}
	}
}

func TestSpecTextIsATemplateOverTheValues(t *testing.T) {
	e, _, _, _ := testEnv()
	units := loadOne(t, e, "[file.a]\npath = \"{{.Dir}}/{{.User}}/{{add 1 2}}\"\nsource = \"f\"\n", map[string]string{"f": "x"})
	if got := headlines(t, units[0].Step); !slices.Contains(got, "create /srv/u/3") {
		t.Errorf("plan = %q, want the file at /srv/u/3", got)
	}
	_, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": "[file.a]\npath = \"{{.Home}}\"\n"}))
	if err == nil || !strings.Contains(err.Error(), "Home") {
		t.Errorf("load = %v, want an error that names the missing value Home", err)
	}
}

func TestClosureAddsWhatAUnitWaitsOn(t *testing.T) {
	e, _, _, _ := testEnv()
	spec := "[file.a]\npath = \"/a\"\nsource = \"f\"\nafter = \"file/b\"\n" +
		"[file.b]\npath = \"/b\"\nsource = \"f\"\nafter = \"file/c\"\n" +
		"[file.c]\npath = \"/c\"\nsource = \"f\"\n[file.d]\npath = \"/d\"\nsource = \"f\"\n"
	units := loadOne(t, e, spec, map[string]string{"f": ""})
	got, err := Closure(units, []string{"file/d", "file/a"})
	if want := []string{"file/c", "file/b", "file/a", "file/d"}; err != nil || !slices.Equal(Names(got), want) {
		t.Errorf("closure = %q, %v; want %q", Names(got), err, want)
	}
}

func TestClosureRefusesAnUnknownUnit(t *testing.T) {
	_, err := Closure(nil, []string{"file/x"})
	var unknown *UnknownUnit
	if !errors.As(err, &unknown) || unknown.Name != "file/x" {
		t.Errorf("closure = %v, want an *UnknownUnit for file/x", err)
	}
}

func TestEveryKindFieldIsAKind(t *testing.T) {
	kind := reflect.TypeFor[unitKind]()
	for f := range reflect.TypeFor[spec]().Fields() {
		if f.Type.Kind() != reflect.Map || f.Type.Key().Kind() != reflect.String || !f.Type.Elem().Implements(kind) {
			t.Errorf("spec.%s is %s, want a map of string to a unitKind", f.Name, f.Type)
		}
		if tag := f.Tag.Get("toml"); tag == "" || tag != strings.ToLower(tag) {
			t.Errorf("spec.%s has the toml tag %q, want the kind's name", f.Name, tag)
		}
	}
}

func TestSelfIsEnvSelf(t *testing.T) {
	e, m, _, _ := testEnv()
	m.Dirs["/etc"] = 0o755
	e.Self = "/opt/ws/bin/ws"
	units := loadOne(t, e, "[file.a]\npath = \"/etc{{ self }}\"\nsource = \"a\"\n", map[string]string{"a": "x"})
	if got := headlines(t, units[0].Step); len(got) == 0 || !strings.Contains(got[len(got)-1], "/etc/opt/ws/bin/ws") {
		t.Errorf("plan = %q, want the file at /etc/opt/ws/bin/ws", got)
	}
}

func TestSelfWithoutAPathFailsTheLoad(t *testing.T) {
	e, _, _, _ := testEnv()
	_, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": "[file.a]\npath = \"{{ self }}\"\nsource = \"a\"\n", "r/a": "x"}))
	if err == nil || !strings.Contains(err.Error(), "self: the path of the running ws is unknown") {
		t.Errorf("load = %v, want self's error", err)
	}
}
