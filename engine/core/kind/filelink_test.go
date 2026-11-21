package kind

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// linkSpec is a link unit at /usr/local/bin/x to /opt/t/x, with the extra fields.
func linkSpec(extra string) string {
	return "[file.x]\npath = \"/usr/local/bin/x\"\nlink = \"/opt/t/x\"\n" + extra
}

// withBin gives the machine /usr/local/bin and the directories above it.
func withBin(m *effecttest.Machine) {
	for _, d := range []string{"/usr", "/usr/local", "/usr/local/bin"} {
		m.Dirs[d] = 0o755
	}
}

func TestFileLinkPlansTheLink(t *testing.T) {
	e, m, _, _ := testEnv(false)
	withBin(m)
	units := loadOne(t, e, linkSpec(""), nil)
	if got := headlines(t, units[0].Step); !slices.Equal(got, []string{"link /usr/local/bin/x to /opt/t/x"}) {
		t.Errorf("plan = %q, want the link alone", got)
	}
	if err := run(t, e, units); err != nil {
		t.Fatal(err)
	}
	if got := m.Links["/usr/local/bin/x"]; got != "/opt/t/x" {
		t.Errorf("link = %q, want /opt/t/x", got)
	}
}

func TestFileLinkAgainChangesNothing(t *testing.T) {
	e, m, _, _ := testEnv(false)
	withBin(m)
	m.Links["/usr/local/bin/x"] = "/opt/t/x"
	if got := headlines(t, loadOne(t, e, linkSpec(""), nil)[0].Step); got != nil {
		t.Errorf("plan = %q, want none", got)
	}
}

func TestFileLinkRepointsAWrongLink(t *testing.T) {
	e, m, _, _ := testEnv(false)
	withBin(m)
	m.Links["/usr/local/bin/x"] = "/opt/old/x"
	units := loadOne(t, e, linkSpec(""), nil)
	if err := run(t, e, units); err != nil {
		t.Fatal(err)
	}
	if got := m.Links["/usr/local/bin/x"]; got != "/opt/t/x" {
		t.Errorf("link = %q, want /opt/t/x", got)
	}
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
}

func TestFileLinkMakesItsParents(t *testing.T) {
	e, m, _, _ := testEnv(false)
	if err := run(t, e, loadOne(t, e, linkSpec(""), nil)); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{"/usr", "/usr/local", "/usr/local/bin"} {
		if m.Dirs[d] != 0o755 {
			t.Errorf("%s = %o, want 755", d, m.Dirs[d])
		}
	}
	if m.Links["/usr/local/bin/x"] != "/opt/t/x" {
		t.Errorf("links = %v, want /usr/local/bin/x", m.Links)
	}
}

func TestFileLinkTakesNoContent(t *testing.T) {
	e, _, _, _ := testEnv(false)
	alone := "a link takes a path, a target, after and when alone"
	for extra, want := range map[string]string{
		"template = \"f\"\n":         alone,
		"source = \"f\"\n":           alone,
		"absent = true\n":            alone,
		"line = \"x\"\n":             alone,
		"has = [\"/bin/true\"]\n":    alone,
		"secret = true\n":            alone,
		"check = [\"/bin/true\"]\n":  alone,
		"mode = 0o755\n":             alone,
		"group_of = \"u\"\n":         alone,
		"group = \"g\"\n":            alone,
		"ours = true\n":              alone,
		"backup = true\n":            alone,
		"absent_when_empty = true\n": alone,
		"seed = true\n":              alone,
		"owner = \"u\"\n":            "a link is root's",
		"service = \"x.service\"\n":  "a link takes no service",
	} {
		_, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": linkSpec(extra), "r/f": ""}))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("load with %q = %v, want %q", extra, err, want)
		}
	}
	for _, target := range []string{"t/x", "/opt/../t/x", "/opt/t/"} {
		spec := "[file.x]\npath = \"/usr/local/bin/x\"\nlink = \"" + target + "\"\n"
		if _, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": spec})); err == nil || !strings.Contains(err.Error(), "not a clean absolute path") {
			t.Errorf("load of the target %q = %v, want a refusal", target, err)
		}
	}
	if _, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": linkSpec("owner = \"root\"\n")})); err != nil {
		t.Errorf("load of a link that names root = %v, want none", err)
	}
}
