package kind

import (
	"strings"
	"testing"
)

func TestLoadRefusesATemplateWithoutTheEnding(t *testing.T) {
	e, _, _, _ := testEnv()
	specs := map[string]string{
		"file":     "[file.a]\npath = \"/x\"\ntemplate = \"%s\"\n",
		"packages": "[packages.a]\ntemplate = \"%s\"\n",
	}
	names := map[string]bool{
		"a.tmpl": true, "files/a.conf.tmpl": true, "a": false, "a.conf": false, "a.toml": false,
		"a.tmpl.bak": false, "tmpl": false, "files/a.tmpl/b": false,
	}
	for kind, spec := range specs {
		for name, ok := range names {
			files := map[string]string{"r/units.toml.tmpl": strings.Replace(spec, "%s", name, 1), "r/" + name: "x"}
			_, err := Load(e, tree(files))
			switch {
			case ok && err != nil:
				t.Errorf("%s template %q: load = %v, want none", kind, name, err)
			case !ok && (err == nil || !strings.Contains(err.Error(), "ends in .tmpl")):
				t.Errorf("%s template %q: load = %v, want an error that names the .tmpl rule", kind, name, err)
			}
		}
	}
}
