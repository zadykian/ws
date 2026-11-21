package kind

import (
	"strings"
	"testing"
)

const groupSpec = "[file.a]\npath = \"/etc/n/a.conf\"\nmode = 0o640\ngroup = \"net\"\nsource = \"f\"\n"

func TestFileGroupIsTheGroupsGID(t *testing.T) {
	e, m, _, _ := testEnv(false)
	m.Files["/etc/group"] = []byte("root:x:0:\nnet:x:998:\n")
	units := loadOne(t, e, groupSpec, map[string]string{"f": "x\n"})
	if err := run(t, e, units); err != nil {
		t.Fatal(err)
	}
	if got := m.Owners["/etc/n/a.conf"]; got != [2]int{0, 998} {
		t.Errorf("owner = %v, want root:998", got)
	}
	if got := headlines(t, units[0].Step); len(got) != 0 {
		t.Errorf("second plan = %q, want nothing", got)
	}
}

func TestFileGroupMissingFailsThePlan(t *testing.T) {
	e, m, _, _ := testEnv(false)
	m.Files["/etc/group"] = []byte("root:x:0:\n")
	units := loadOne(t, e, groupSpec, map[string]string{"f": "x\n"})
	if _, err := units[0].Step.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "the group net is missing") {
		t.Errorf("plan = %v, want the group missing", err)
	}
	if _, written := m.Files["/etc/n/a.conf"]; written {
		t.Error("the file was written")
	}
}

func TestFileGroupRefusedAtLoad(t *testing.T) {
	for spec, want := range map[string]string{
		"owner = \"u\"\npath = \"/home/u/a\"\ngroup = \"net\"\nsource = \"f\"\n": "group takes a file of root's",
		"path = \"/etc/a\"\ngroup = \"net\"\nabsent = true\n":                    "an absent file takes",
		"path = \"/etc/a\"\ngroup = \"a b\"\nsource = \"f\"\n":                   "no user's name",
	} {
		e, _, _, _ := testEnv(false)
		_, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": "[file.a]\n" + spec, "r/f": ""}))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: load = %v, want %q", spec, err, want)
		}
	}
}
