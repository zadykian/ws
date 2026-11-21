package kind

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

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

func TestBackupKeepsTheOldFile(t *testing.T) {
	e, m, _, _ := testEnv(false)
	withAccount(m)
	m.Files["/home/u/a.yml"] = []byte("by hand\n")
	units := loadOne(t, e, "[file.a]\npath = \"/home/u/a.yml\"\nowner = \"u\"\nsource = \"a\"\nbackup = true\n",
		map[string]string{"a": "new\n"})
	if err := run(t, e, units); err != nil {
		t.Fatal(err)
	}
	var kept []string
	for p, data := range m.Files {
		if strings.HasPrefix(p, "/home/u/a.yml.") && strings.HasSuffix(p, "~") {
			kept = append(kept, string(data))
		}
	}
	if string(m.Files["/home/u/a.yml"]) != "new\n" || !slices.Equal(kept, []string{"by hand\n"}) {
		t.Errorf("file = %q, copies = %q; want the new file and one copy of the old", m.Files["/home/u/a.yml"], kept)
	}
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
}

func TestBackupOfAMissingFileKeepsNothing(t *testing.T) {
	e, m, _, _ := testEnv(false)
	withAccount(m)
	units := loadOne(t, e, "[file.a]\npath = \"/home/u/a.yml\"\nowner = \"u\"\nsource = \"a\"\nbackup = true\n",
		map[string]string{"a": "new\n"})
	if err := run(t, e, units); err != nil {
		t.Fatal(err)
	}
	for p := range m.Files {
		if strings.HasSuffix(p, "~") {
			t.Errorf("copy %s of a file that was not there", p)
		}
	}
}

func TestBackupRefusesSecretAbsentOrLine(t *testing.T) {
	e, _, _, _ := testEnv(false)
	for _, spec := range []string{
		"path = \"/a\"\nsource = \"f\"\nsecret = true\nbackup = true\n",
		"path = \"/a\"\nabsent = true\nbackup = true\n",
		"path = \"/a\"\nline = \". /x\"\nbackup = true\n",
	} {
		_, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": "[file.a]\n" + spec, "r/f": ""}))
		if err == nil || !strings.Contains(err.Error(), "backup takes a file that ws writes whole") {
			t.Errorf("load of\n%s= %v, want backup refused", spec, err)
		}
	}
}

// An account's file where the account is root's names root, and below root's home it refuses a link on
// the way as another account's does. A file of root's that names root lies anywhere.
func TestFileNamingRootRefusesALinkInRootsHome(t *testing.T) {
	e, m, _, _ := testEnv(false)
	m.Results["getent passwd root"] = effecttest.Result{Out: []byte("root:x:0:0:root:/root:/bin/bash\n")}
	m.Dirs["/root"] = 0o700
	m.Links["/root/.config"] = "/etc"
	home := loadOne(t, e, "[file.a]\npath = \"/root/.config/a.yml\"\nowner = \"root\"\nsource = \"a\"\nbackup = true\n",
		map[string]string{"a": "new\n"})
	if err := run(t, e, home); err == nil || !strings.Contains(err.Error(), "/root/.config is a link to /etc") {
		t.Errorf("run = %v, want the link refused", err)
	}
	if _, ok := m.Files["/etc/a.yml"]; ok {
		t.Error("file written through the link")
	}
	outside := loadOne(t, e, "[file.a]\npath = \"/etc/x/a.conf\"\nowner = \"root\"\nsource = \"a\"\n",
		map[string]string{"a": "new\n"})
	if err := run(t, e, outside); err != nil || string(m.Files["/etc/x/a.conf"]) != "new\n" {
		t.Errorf("run = %v, file %q; want root's file outside its home written", err, m.Files["/etc/x/a.conf"])
	}
}
