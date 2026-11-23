package kind

import (
	"archive/tar"
	"bytes"
	"crypto/sha256"
	"fmt"
	"slices"
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
