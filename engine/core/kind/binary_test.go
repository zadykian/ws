package kind

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

const binaryPath = "/usr/local/sbin/x"

// binarySpecText pins a 1.2 whose amd64 file is the body "bin", and whose arm64 file is "arm".
func binarySpecText(extra string) string {
	return fmt.Sprintf("[binary.x]\nversion = \"1.2\"\nurl = \"https://x/v{version}/x-{arch}\"\npath = %q\n"+
		"sha256 = { amd64 = \"%x\", arm64 = \"%x\" }\n%s", binaryPath, sha256.Sum256([]byte("bin")), sha256.Sum256([]byte("arm")), extra)
}

// binaryMachine serves the bodies of both architectures, on a machine of dpkg's arch.
func binaryMachine(m *effecttest.Machine, arch, body string) {
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte(arch + "\n")}
	m.Pages["https://x/v1.2/x-amd64"] = []byte(body)
	m.Pages["https://x/v1.2/x-arm64"] = []byte("arm")
	m.Pages["https://x/v1.2/x-x86_64"] = []byte(body)
}

func TestBinaryDownloadsByTheArchsSum(t *testing.T) {
	e, m, _, _ := testEnv()
	binaryMachine(m, "arm64", "bin")
	units := loadOne(t, e, binarySpecText(""), nil)
	if err := applyOnce(t, units[0].Step); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(m.Calls, "download https://x/v1.2/x-arm64") || string(m.Files[binaryPath]) != "arm" {
		t.Errorf("calls = %q, file %q, want the arm64 file", m.Calls, m.Files[binaryPath])
	}
}

func TestBinaryWrongSumLeavesNoFile(t *testing.T) {
	e, m, _, _ := testEnv()
	binaryMachine(m, "amd64", "not the pinned file")
	err := applyOnce(t, loadOne(t, e, binarySpecText(""), nil)[0].Step)
	if err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Errorf("apply = %v, want a wrong SHA-256", err)
	}
	if _, ok := m.Files[binaryPath]; ok {
		t.Errorf("file %s exists after a refused download", binaryPath)
	}
}

func TestBinaryCurrentPlansNothing(t *testing.T) {
	e, m, _, _ := testEnv()
	binaryMachine(m, "amd64", "bin")
	units := loadOne(t, e, binarySpecText(""), nil)
	if got := headlines(t, units[0].Step); len(got) == 0 {
		t.Fatal("plan of a missing file is empty, want a download")
	}
	if err := applyOnce(t, units[0].Step); err != nil {
		t.Fatal(err)
	}
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("plan of the current file = %q, want none", got)
	}
}

func TestBinaryArchNamesSpellTheURL(t *testing.T) {
	e, m, _, _ := testEnv()
	binaryMachine(m, "amd64", "bin")
	units := loadOne(t, e, binarySpecText("arch_names = { amd64 = \"x86_64\" }\n"), nil)
	if err := applyOnce(t, units[0].Step); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(m.Calls, "download https://x/v1.2/x-x86_64") {
		t.Errorf("calls = %q, want the asset spelled x86_64", m.Calls)
	}
}

func TestBinaryAllForEveryArch(t *testing.T) {
	e, m, _, _ := testEnv()
	m.Pages["https://x/v1.2/x-all"] = []byte("bin")
	spec := fmt.Sprintf("[binary.x]\nversion = \"1.2\"\nurl = \"https://x/v{version}/x-{arch}\"\npath = %q\nsha256 = { all = \"%x\" }\n",
		binaryPath, sha256.Sum256([]byte("bin")))
	if err := applyOnce(t, loadOne(t, e, spec, nil)[0].Step); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(m.Calls, "run dpkg --print-architecture") || string(m.Files[binaryPath]) != "bin" {
		t.Errorf("calls = %q, want the shared file with no architecture read", m.Calls)
	}
}

func TestBinaryModeDefaults755(t *testing.T) {
	for extra, want := range map[string]uint32{"": 0o755, "mode = 0o700\n": 0o700} {
		e, m, _, _ := testEnv()
		binaryMachine(m, "amd64", "bin")
		if err := applyOnce(t, loadOne(t, e, binarySpecText(extra), nil)[0].Step); err != nil {
			t.Fatal(err)
		}
		got, err := m.RootAttrs(binaryPath)
		if err != nil || uint32(got.Mode) != want || got.UID != 0 {
			t.Errorf("attrs with %q = %+v, %v; want mode %o, root's", extra, got, err, want)
		}
	}
}

func TestBinaryRefusesARelativePath(t *testing.T) {
	e, _, _, _ := testEnv()
	for _, p := range []string{"bin/x", "/usr/../x", "/", "/usr/local/"} {
		spec := strings.Replace(binarySpecText(""), binaryPath, p, 1)
		if _, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": spec})); err == nil || !strings.Contains(err.Error(), "clean absolute path") {
			t.Errorf("load with path %q = %v, want a clean absolute path asked", p, err)
		}
	}
}

func TestBinaryRefusesABadSum(t *testing.T) {
	e, _, _, _ := testEnv()
	for spec, want := range map[string]string{
		"[binary.x]\nversion = \"1\"\nurl = \"u\"\npath = \"/x\"\n":                             "a sha256 for each",
		"[binary.x]\nversion = \"1\"\nurl = \"u\"\npath = \"/x\"\nsha256 = { amd64 = \"X\" }\n": "64 lowercase hex",
		binarySpecText("mode = 0o1777\n"):                                                       "permission bits",
	} {
		if _, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": spec})); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("load of\n%s= %v, want %q", spec, err, want)
		}
	}
}
