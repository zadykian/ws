package kind

import (
	"slices"
	"strings"
	"testing"
)

// sealedSpec is a secret file of root's whose template is a.tmpl.
const sealedSpec = "[file.a]\npath = \"/etc/a.env\"\nmode = 0o600\ntemplate = \"a.tmpl\"\nsecret = true\n"

// envSpec is a secret env file of root's that goes where its template fills to white space.
const envSpec = "[file.env]\npath = \"/etc/ws/env\"\nmode = 0o640\ntemplate = \"env.tmpl\"\nsecret = true\n" +
	"ours = true\nabsent_when_empty = true\n"

// envTemplate writes A of /etc/ws/inputs, under ws's mark, where the file sets it.
const envTemplate = `{{ with envvar "/etc/ws/inputs" "A" }}` + oursMark + "\nexport A={{ . }}{{ end }}\n"

func TestSecretTemplateFillsWithTheFactsAlone(t *testing.T) {
	e, m, _, _ := testEnv(false)
	m.Dirs["/etc"], m.Dirs["/etc/ws"] = 0o755, 0o755
	m.Files["/etc/ws/env"] = []byte("export T='hunter2'\n")
	units := loadOne(t, e, sealedSpec, map[string]string{"a.tmpl": `T={{ envvar "/etc/ws/env" "T" }}` + "\n"})
	p, err := units[0].Step.Plan(t.Context())
	if err != nil || len(p.Changes) != 1 {
		t.Fatalf("plan = %v, %v; want one change", p.Changes, err)
	}
	if s := p.Changes[0].String(); strings.Contains(s, "hunter2") {
		t.Errorf("change = %q, want no value", s)
	}
	if err := run(t, e, units); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/etc/a.env"]); got != "T=hunter2\n" {
		t.Errorf("file = %q, want the value filled", got)
	}
	units = loadOne(t, e, sealedSpec, map[string]string{"a.tmpl": "{{ (secrets).Env }}\n"})
	if _, err := units[0].Step.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), `"secrets" not defined`) {
		t.Errorf("plan = %v, want the func unknown", err)
	}
}

func TestAbsentWhenEmptyRemovesTheFile(t *testing.T) {
	e, m, _, _ := testEnv(false)
	m.Files["/etc/ws/inputs"] = []byte("B=1\n")
	m.Files["/etc/ws/env"] = []byte(oursMark + "\nexport A=hunter2\n")
	units := loadOne(t, e, envSpec, map[string]string{"env.tmpl": envTemplate})
	if got := headlines(t, units[0].Step); !slices.Equal(got, []string{"remove /etc/ws/env"}) {
		t.Errorf("plan = %q, want the removal", got)
	}
	if err := run(t, e, units); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Files["/etc/ws/env"]; ok {
		t.Error("the file stayed")
	}
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
	if _, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": "[file.a]\npath = \"/a\"\nsource = \"f\"\nabsent_when_empty = true\n", "r/f": ""})); err == nil ||
		!strings.Contains(err.Error(), "absent_when_empty goes with template") {
		t.Errorf("load of a source = %v, want it refused", err)
	}
}

func TestSecretFileTakesACheck(t *testing.T) {
	e, _, _, _ := testEnv(false)
	units := loadOne(t, e, "[file.a]\npath = \"/etc/a.env\"\nsecret = true\ncheck = [\"c\", \"{}\"]\nsource = \"f\"\n", map[string]string{"f": "TOKEN=1\n"})
	if len(units) != 1 {
		t.Fatalf("units = %d, want 1", len(units))
	}
}
