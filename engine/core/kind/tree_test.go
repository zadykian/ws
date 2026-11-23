package kind

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// treeTar is an archive of one file, ./f, that holds body.
func treeTar(t *testing.T, body string) []byte {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if err := tw.WriteHeader(&tar.Header{Name: "./f", Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// treeText is tree/t at /opt/t, of the archives 1.0 and 2.0 of arch, which m serves with their
// version as f.
func treeText(t *testing.T, m *effecttest.Machine, arch, spelled string) string {
	text := "[tree.t]\npath = \"/opt/t\"\narch_names = { amd64 = \"x64\" }\n"
	for _, v := range []string{"1.0", "2.0"} {
		body := treeTar(t, v)
		m.Pages["https://x/t-"+v+"-"+spelled+".tar"] = body
		text += fmt.Sprintf("[[tree.t.archive]]\nversion = %q\nurl = \"https://x/t-{version}-{arch}.tar\"\nsha256 = { %s = \"%x\" }\n",
			v, arch, sha256.Sum256(body))
	}
	return text
}

func TestTreeUnpacksItsArchivesByDpkgsArch(t *testing.T) {
	e, m, _, _ := testEnv()
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	units := loadOne(t, e, treeText(t, m, "amd64", "x64"), nil)
	if err := applyOnce(t, units[0].Step); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"download https://x/t-1.0-x64.tar", "download https://x/t-2.0-x64.tar"} {
		if !slices.Contains(m.Calls, want) {
			t.Errorf("calls = %q, want %s", m.Calls, want)
		}
	}
	if got := string(m.Files["/opt/t/f"]); got != "2.0" {
		t.Errorf("f = %q, want the later archive's 2.0", got)
	}
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("plan of the current tree = %q, want none", got)
	}
}

func TestTreeRefusesABadSpec(t *testing.T) {
	sum := strings.Repeat("a", 64)
	archive := "[[tree.t.archive]]\nversion = \"1\"\nurl = \"https://x/t.tar\"\nsha256 = { all = \"" + sum + "\" }\n"
	for spec, want := range map[string]string{
		"[tree.t]\npath = \"/usr\"\n" + archive:               "not a clean absolute path below a directory of /",
		"[tree.t]\npath = \"opt/t\"\n" + archive:              "not a clean absolute path",
		"[tree.t]\npath = \"/opt/t\"\nstrip = -1\n" + archive: "strip is below 0",
		"[tree.t]\npath = \"/opt/t\"\n":                       "an archive at least",
		"[tree.t]\npath = \"/opt/t\"\n[[tree.t.archive]]\nurl = \"https://x/t.tar\"\nsha256 = { all = \"" + sum + "\" }\n":        "takes a version",
		"[tree.t]\npath = \"/opt/t\"\n[[tree.t.archive]]\nversion = \"1\"\nurl = \"https://x/t.tar\"\nsha256 = { all = \"A\" }\n": "64 lowercase hex",
	} {
		e, _, _, _ := testEnv()
		if _, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": spec})); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("load of %q = %v, want %q", spec, err, want)
		}
	}
}
