package kind

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
)

func TestGroupOfTakesThePrimaryGroup(t *testing.T) {
	e, m, _, _ := testEnv(false)
	withAccount(m)
	m.Dirs["/etc"] = 0o755
	units := loadOne(t, e, "[file.a]\npath = \"/etc/a.env\"\nmode = 0o640\nsource = \"a\"\ngroup_of = \"u\"\n", map[string]string{"a": "x\n"})
	if err := run(t, e, units); err != nil {
		t.Fatal(err)
	}
	if got, err := m.RootAttrs("/etc/a.env"); err != nil || got != (effect.Attrs{GID: 1001, Mode: 0o640}) {
		t.Errorf("attrs = %+v, %v; want root's with u's group, 640", got, err)
	}
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
}

func TestGroupOfRefusesAnotherOwner(t *testing.T) {
	e, _, _, _ := testEnv(false)
	for spec, want := range map[string]string{
		"path = \"/home/u/a\"\nowner = \"u\"\nsource = \"f\"\ngroup_of = \"u\"\n": "group_of takes a file of root's",
		"path = \"/a\"\nsource = \"f\"\ngroup_of = \"u x\"\n":                     "is no user's name",
		"path = \"/a\"\nabsent = true\ngroup_of = \"u\"\n":                        "an absent file takes a path, an owner and ours alone",
		"path = \"/home/u/a\"\nowner = \"u\"\nsource = \"f\"\nours = true\n":      "ours takes a file of root's",
	} {
		_, err := Load(e, tree(map[string]string{"r/units.toml": "[file.a]\n" + spec, "r/f": ""}))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("load of\n%s= %v, want %q", spec, err, want)
		}
	}
}

func TestOursRefusesAFileByHandInADryRunToo(t *testing.T) {
	e, m, _, _ := testEnv(false)
	m.Files["/etc/a.env"] = []byte("TOKEN=by-hand\n")
	units := loadOne(t, e, "[file.a]\npath = \"/etc/a.env\"\nsource = \"a\"\nsecret = true\nours = true\n", map[string]string{"a": oursMark + "\nTOKEN=new\n"})
	_, err := units[0].Step.Plan(t.Context())
	if err == nil || !strings.Contains(err.Error(), "/etc/a.env lacks the line '# Written by github.com/zadykian/ws' at its top") {
		t.Fatalf("plan = %v, want the file by hand refused", err)
	}
	if strings.Contains(err.Error(), "TOKEN") || strings.Contains(err.Error(), "by-hand") {
		t.Errorf("error = %q, want no byte of the file", err)
	}
	if run(t, e, units) == nil || string(m.Files["/etc/a.env"]) != "TOKEN=by-hand\n" {
		t.Errorf("file = %q, want it left", m.Files["/etc/a.env"])
	}
}

func TestOursTakesAMissingFile(t *testing.T) {
	e, m, _, _ := testEnv(false)
	m.Dirs["/etc"] = 0o755
	units := loadOne(t, e, "[file.a]\npath = \"/etc/a.env\"\nsource = \"a\"\nours = true\n", map[string]string{"a": oursMark + "\nA=1\n"})
	if err := run(t, e, units); err != nil {
		t.Fatal(err)
	}
	m.Files["/etc/a.env"] = []byte(oursMark + ", from an older run.\nA=0\n")
	if got := headlines(t, units[0].Step); !slices.Equal(got, []string{"write /etc/a.env"}) {
		t.Errorf("plan = %q, want ws's own file replaced", got)
	}
}

func TestOursKeepsAFileByHandFromRemoval(t *testing.T) {
	e, m, _, _ := testEnv(false)
	m.Files["/etc/a.env"] = []byte("A=1\n")
	units := loadOne(t, e, "[file.a]\npath = \"/etc/a.env\"\nabsent = true\nours = true\n", nil)
	if err := run(t, e, units); err == nil || string(m.Files["/etc/a.env"]) != "A=1\n" {
		t.Errorf("run = %v, file %q; want the file by hand refused and kept", err, m.Files["/etc/a.env"])
	}
}
