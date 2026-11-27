package kind

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// partFields lists each field of a file part by its TOML tag, with text that sets it. takes names
// the modes of partModes whose part takes the field, and a service's file is the mode that
// checkService leaves alone. A new field fails the test that ranges over the tags until a line
// here names it.
var partFields = map[string]struct {
	toml  string
	takes string
}{
	"path":              {"path = \"/root/.rc\"\n", "line link absent seed service"},
	"mode":              {"mode = 0o600\n", "line seed service"},
	"owner":             {"owner = \"root\"\n", "line link absent seed service"},
	"group":             {"group = \"g\"\n", "seed service"},
	"template":          {"template = \"f.tmpl\"\n", "seed service"},
	"source":            {"source = \"f\"\n", "seed service"},
	"check":             {"check = [\"/bin/true\"]\n", "line service"},
	"secret":            {"secret = true\n", "seed service"},
	"absent":            {"absent = true\n", "absent service"},
	"line":              {"line = \"x\"\n", "line"},
	"has":               {"has = [\"/bin/true\"]\n", "line service"},
	"ours":              {"ours = true\n", "absent service"},
	"backup":            {"backup = true\n", "service"},
	"absent_when_empty": {"absent_when_empty = true\n", "service"},
	"link":              {"link = \"/opt/x\"\n", "link"},
	"line_at":           {"line_at = \"end\"\n", "line service"},
	"mark":              {"ours = true\nmark = \"//\"\n", "absent service"},
	"keep_foreign":      {"ours = true\nkeep_foreign = true\n", "absent service"},
	"seed":              {"seed = true\n", "service"},
}

// partModes are the modes of a file part that refuse fields. base makes the part, own lists the
// tags it sets, and content gives it a source where the field is no content, as a seed needs one.
var partModes = []struct {
	name, base string
	own        []string
	content    bool
}{
	{"line", "path = \"/root/.profile\"\nowner = \"root\"\nline = \". /etc/x\"\n", []string{"path", "owner", "line"}, false},
	{"link", "path = \"/root/.l\"\nlink = \"/opt/x\"\n", []string{"path", "link"}, false},
	{"absent", "path = \"/root/.a\"\nabsent = true\n", []string{"path", "absent"}, false},
	{"seed", "path = \"/root/.s\"\nseed = true\n", []string{"path", "seed"}, true},
}

// tags returns the TOML tag of each field of t, those of an embedded struct too.
func tags(t reflect.Type) []string {
	var all []string
	for f := range t.Fields() {
		if f.Anonymous {
			all = append(all, tags(f.Type)...)
		} else if tag, ok := f.Tag.Lookup("toml"); ok {
			all = append(all, tag)
		}
	}
	return all
}

func TestEveryFieldOfAPartIsListed(t *testing.T) {
	for _, tag := range tags(reflect.TypeFor[filePart]()) {
		if _, ok := partFields[tag]; !ok {
			t.Errorf("field %q is not in partFields: say whether a line and a service's file take it", tag)
		}
	}
}

func TestEachModeRefusesEveryFieldOutsideItsList(t *testing.T) {
	e, _, _, _ := testEnv()
	for _, mode := range partModes {
		for _, tag := range tags(reflect.TypeFor[filePart]()) {
			field := partFields[tag]
			if slices.Contains(mode.own, tag) {
				continue
			}
			spec := mode.base + field.toml
			if mode.content && tag != "template" && tag != "source" {
				spec += "source = \"f\"\n"
			}
			_, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": "[file.a]\n" + spec, "r/f": "", "r/f.tmpl": ""}))
			taken := slices.Contains(strings.Fields(field.takes), mode.name)
			switch {
			case taken && err != nil:
				t.Errorf("mode %s with %s: load = %v, want none", mode.name, tag, err)
			case !taken && err == nil:
				t.Errorf("mode %s with %s: load succeeded, want a refusal", mode.name, tag)
			}
		}
	}
}

func TestServiceCheckRefusesALineAndALink(t *testing.T) {
	for _, tag := range tags(reflect.TypeFor[filePart]()) {
		var part filePart
		if err := toml.Unmarshal([]byte(partFields[tag].toml), &part); err != nil {
			t.Fatalf("%s: %v", tag, err)
		}
		taken := slices.Contains(strings.Fields(partFields[tag].takes), "service")
		if err := part.checkService(); taken != (err == nil) {
			t.Errorf("a service's file with %s: checkService = %v, want taken = %v", tag, err, taken)
		}
	}
}

func TestServiceFileRefusesALineAndALink(t *testing.T) {
	e, _, _, _ := testEnv()
	const base = "[service.a]\nunit = \"a.service\"\n[[service.a.file]]\npath = \"/root/.a\"\nowner = \"root\"\n"
	for _, tag := range []string{"line", "link"} {
		_, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": base + partFields[tag].toml, "r/f": ""}))
		if err == nil || !strings.Contains(err.Error(), "takes no service") {
			t.Errorf("a service's file with %s: load = %v, want it to take no service", tag, err)
		}
	}
}

func TestSeedStaysAllowedBesideAService(t *testing.T) {
	e, _, _, _ := testEnv()
	const file = "path = \"/etc/a.json\"\nsource = \"f\"\nseed = true\n"
	for name, spec := range map[string]string{
		"a file with a service": "[file.a]\n" + file + "service = \"a.service\"\n",
		"a service's file":      "[service.a]\nunit = \"a.service\"\n[[service.a.file]]\n" + file,
	} {
		if _, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": spec, "r/f": ""})); err != nil {
			t.Errorf("%s with a seed: load = %v, want none", name, err)
		}
	}
}
