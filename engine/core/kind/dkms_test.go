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
	"github.com/zadykian/ws/engine/core/resource/fetch"
)

const (
	dkmsURL    = "https://x/m/archive/abc.tar.gz"
	dkmsKernel = "7.0.0-1-generic"
	// dkmsConf is the role's dkms.conf, which the unit writes in place of the tarball's.
	dkmsConf = "PACKAGE_NAME=\"m\"\nPACKAGE_VERSION=\"{version}\"\n"
)

// dkmsTarball is a host's tarball of a commit of m: one directory, with a dkms.conf of a fixed 1.
func dkmsTarball(t *testing.T) []byte {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for name, body := range map[string]string{"m-abc/": "", "m-abc/dkms.conf": "PACKAGE_VERSION=\"1.0\"\n", "m-abc/m.c": "x"} {
		hdr := &tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(body))}
		if strings.HasSuffix(name, "/") {
			hdr.Typeflag, hdr.Mode = tar.TypeDir, 0o755
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// dkmsText is the spec of dkms/a, the module m at version 1, pinned to the sum of body. A field of
// over replaces its own, and an empty one leaves it out.
func dkmsText(body []byte, over map[string]string) string {
	fields := map[string]string{"module": `"m"`, "version": `"1"`, "url": `"` + dkmsURL + `"`,
		"sha256": fmt.Sprintf("%q", fmt.Sprintf("%x", sha256.Sum256(body))), "strip": "1", "conf": `"dkms.conf"`}
	for k, v := range over {
		fields[k] = v
	}
	text := "[dkms.a]\n"
	for _, k := range []string{"module", "version", "url", "sha256", "strip", "conf", "load", "unload", "hint"} {
		if fields[k] != "" {
			text += k + " = " + fields[k] + "\n"
		}
	}
	return text
}

// dkmsMachine serves the tarball and runs the kernel, with no dkms installed yet.
func dkmsMachine(t *testing.T) (Env, *effecttest.Machine, *effecttest.Out, []byte) {
	e, m, _, out := testEnv()
	body := dkmsTarball(t)
	m.Pages[dkmsURL] = body
	m.Results["uname -r"] = effecttest.Result{Out: []byte(dkmsKernel + "\n")}
	return e, m, out, body
}

func TestDkmsRefusesBadFields(t *testing.T) {
	e, _, _, body := dkmsMachine(t)
	for _, tc := range []struct {
		over map[string]string
		want string
	}{
		{map[string]string{"module": ""}, `module "" is no DKMS module's name`},
		{map[string]string{"module": `"-m"`}, `module "-m" is no DKMS module's name`},
		{map[string]string{"version": `"a/b"`}, `version "a/b" is no DKMS version`},
		{map[string]string{"version": `".."`}, `version ".." is no DKMS version`},
		{map[string]string{"url": `"http://x/m.tar.gz"`}, `url "http://x/m.tar.gz" is no https URL`},
		{map[string]string{"sha256": `"abc"`}, "sha256 is not 64 lowercase hex digits"},
		{map[string]string{"strip": "-1"}, "strip -1 is below 0"},
		{map[string]string{"conf": `"none.conf"`}, "none.conf"},
		{map[string]string{"unload": `["x"]`}, "unload and hint go with load"},
		{map[string]string{"hint": `"h"`}, "unload and hint go with load"},
		{map[string]string{"load": `"a b"`}, `load "a b" is no module's name`},
		{map[string]string{"load": `"y"`, "unload": `["-r"]`}, `unload "-r" is no module's name`},
	} {
		_, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": dkmsText(body, tc.over), "r/dkms.conf": dkmsConf}))
		if err == nil || !strings.Contains(err.Error(), "unit dkms/a: ") || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("load with %v = %v, want %q", tc.over, err, tc.want)
		}
	}
}

func TestDkmsFetchesTheSourceTree(t *testing.T) {
	e, m, _, body := dkmsMachine(t)
	units := loadOne(t, e, dkmsText(body, nil), map[string]string{"dkms.conf": dkmsConf})
	got := headlines(t, units[0].Step)
	want := []string{"create directory " + fetch.CacheDir + ", owner 0:0, mode 755",
		"download " + dkmsURL + " to " + fetch.CacheDir + "/abc.tar.gz", "install " + dkmsURL + " in /usr/src/m-1"}
	if len(got) < 3 || !slices.Equal(got[:3], want) {
		t.Errorf("plan = %q, want it to start with %q", got, want)
	}
	if slices.Contains(m.Calls, "run dpkg --print-architecture") {
		t.Errorf("the source read the architecture: %q", m.Calls)
	}
}

func TestDkmsRefusesAWrongSHA256(t *testing.T) {
	e, m, _, body := dkmsMachine(t)
	m.Pages[dkmsURL] = []byte("not the pinned tarball")
	err := applyOnce(t, loadOne(t, e, dkmsText(body, nil), map[string]string{"dkms.conf": dkmsConf})[0].Step)
	if err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Errorf("apply = %v, want a wrong SHA-256", err)
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasPrefix(c, "root dkms ") }) || m.Files["/usr/src/m-1/m.c"] != nil {
		t.Errorf("calls = %q, want DKMS never to see the source", m.Calls)
	}
}

func TestDkmsWritesItsConf(t *testing.T) {
	e, m, _, body := dkmsMachine(t)
	m.Results["dkms add m/1"] = effecttest.Result{}
	m.Results["dkms install m/1 -k "+dkmsKernel] = effecttest.Result{}
	units := loadOne(t, e, dkmsText(body, nil), map[string]string{"dkms.conf": dkmsConf})
	if err := applyOnce(t, units[0].Step); err != nil {
		t.Fatal(err)
	}
	const conf = "/usr/src/m-1/dkms.conf"
	if got := string(m.Files[conf]); got != "PACKAGE_NAME=\"m\"\nPACKAGE_VERSION=\"1\"\n" || string(m.Files["/usr/src/m-1/m.c"]) != "x" {
		t.Errorf("%s = %q, m.c %q; want the role's conf at version 1 beside the source", conf, got, m.Files["/usr/src/m-1/m.c"])
	}
	if mode, set := m.Modes[conf]; set && mode != 0o644 {
		t.Errorf("%s has the mode %o, want 644", conf, mode)
	}
	i := slices.Index(m.Calls, "root dkms add m/1")
	if i < 0 || !slices.Contains(m.Calls[i:], "root dkms install m/1 -k "+dkmsKernel) {
		t.Errorf("calls = %q, want the add, then the build for the kernel", m.Calls)
	}
	m.Files[dkmsBinary] = []byte("x")
	m.Results["dkms status m"] = effecttest.Result{Out: []byte("m/1, " + dkmsKernel + ", x86_64: installed\n")}
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
}

func TestDkmsPlansWhatItLacks(t *testing.T) {
	build := "build and install m/1 for " + dkmsKernel
	for _, tc := range []struct {
		name, status string
		noDKMS       bool
		want         []string
		warned       bool
	}{
		{"nothing yet", "", true, []string{"add m/1 to DKMS", build}, false},
		{"added", "m/1: added\n", false, []string{build}, false},
		{"installed", "m/1, " + dkmsKernel + ", x86_64: installed\n", false, nil, false},
		{"another version", "m/old, " + dkmsKernel + ", x86_64: installed\n", false,
			[]string{"remove m/old from DKMS", "remove /usr/src/m-old", "add m/1 to DKMS", build}, false},
		{"a version of no word's form", "m/../x, " + dkmsKernel + ", x86_64: installed\n", false,
			[]string{"add m/1 to DKMS", build}, true},
	} {
		_, m, out, _ := dkmsMachine(t)
		m.Results["dkms status m"] = effecttest.Result{Out: []byte(tc.status)}
		if !tc.noDKMS {
			m.Files[dkmsBinary] = []byte("x")
		}
		changes, err := dkmsBuild{m: m, out: out, s: dkmsSpec{Module: "m", Version: "1"}}.Plan(t.Context())
		var got []string
		for _, c := range changes {
			got = append(got, c.String())
		}
		if err != nil || !slices.Equal(got, tc.want) || tc.warned != (len(out.Lines) == 1) {
			t.Errorf("%s: plan = %q, %v, says %q; want %q", tc.name, got, err, out.Lines, tc.want)
		}
	}
}

func TestDkmsRemovesAnotherVersionWithItsSource(t *testing.T) {
	_, m, out, _ := dkmsMachine(t)
	m.Files[dkmsBinary] = []byte("x")
	m.Files["/usr/src/m-old/m.c"] = []byte("x")
	m.Results["dkms status m"] = effecttest.Result{Out: []byte("m/old, " + dkmsKernel + ", x86_64: installed\n")}
	for _, c := range []string{"dkms remove m/old --all", "dkms add m/1", "dkms install m/1 -k " + dkmsKernel} {
		m.Results[c] = effecttest.Result{}
	}
	changes, err := dkmsBuild{m: m, out: out, s: dkmsSpec{Module: "m", Version: "1"}}.Plan(t.Context())
	for _, c := range changes {
		if err == nil {
			err = c.Apply(t.Context())
		}
	}
	want := []string{"root dkms remove m/old --all", "root rm -rf -- /usr/src/m-old", "root dkms add m/1"}
	if i := slices.Index(m.Calls, want[0]); err != nil || i < 0 || !slices.Equal(m.Calls[i:i+3], want) {
		t.Errorf("apply = %v, calls %q; want %q", err, m.Calls, want)
	}
}

func TestParseDKMS(t *testing.T) {
	const v, k = "1", dkmsKernel
	for _, tc := range []struct {
		name, out string
		versions  []string
		installed bool
	}{
		{"nothing", "", nil, false},
		{"installed", "m/" + v + ", " + k + ", x86_64: installed\n", []string{v}, true},
		{"added only", "m/" + v + ": added\n", []string{v}, false},
		{"another kernel", "m/" + v + ", 6.8.0-1-generic, x86_64: installed\n", []string{v}, false},
		{"built, not installed", "m/" + v + ", " + k + ", x86_64: built\n", []string{v}, false},
		{"two versions once each", "m/old, " + k + ", x86_64: installed\nm/old, 6.8.0, x86_64: installed\nm/" + v + ": added\n", []string{"old", v}, false},
		{"a longer version is not the pin", "m/" + v + "x, " + k + ", x86_64: installed\n", []string{v + "x"}, false},
		{"another module", "mm/" + v + ", " + k + ", x86_64: installed\n", nil, false},
	} {
		got := parseDKMS(tc.out, "m", v, k)
		if !slices.Equal(got.versions, tc.versions) || got.installed != tc.installed {
			t.Errorf("%s: parseDKMS = %+v, want versions %q installed %v", tc.name, got, tc.versions, tc.installed)
		}
	}
}
