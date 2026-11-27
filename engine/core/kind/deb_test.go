package kind

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource/fetch"
	"github.com/zadykian/ws/engine/core/step"
)

const (
	debFile    = fetch.CacheDir + "/a_1.2_amd64.deb"
	debInstall = "env DEBIAN_FRONTEND=noninteractive apt-get -o DPkg::Lock::Timeout=600 " +
		"-o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold " +
		"install -y --allow-downgrades " + debFile
)

// debSpecText pins a 1.2 to the sum of the body "deb", for amd64.
func debSpecText(extra string) string {
	return fmt.Sprintf("[deb.a]\npackage = \"a\"\nversion = \"1.2\"\nurl = \"https://x/{version}/a_{version}_{arch}.deb\"\n"+
		"sha256 = { amd64 = \"%x\" }\n%s", sha256.Sum256([]byte("deb")), extra)
}

// debMachine serves body at the pinned URL on an amd64 machine where dpkg knows no a.
func debMachine(m *effecttest.Machine, body string) {
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	m.Packages = effecttest.Packages{}
	m.Pages["https://x/1.2/a_1.2_amd64.deb"] = []byte(body)
	m.Results["dpkg-deb -f "+debFile+" Package Version"] = effecttest.Result{Out: []byte("Package: a\nVersion: 1.2\n")}
	m.Results[debInstall] = effecttest.Result{}
}

// applyOnce plans the step and applies that plan, as a run does before it plans again.
func applyOnce(t *testing.T, s step.Step) error {
	p, err := s.Plan(t.Context())
	if err != nil {
		return err
	}
	return s.Apply(t.Context(), p)
}

func TestDebInstallsThePinnedFile(t *testing.T) {
	e, m, _, _ := testEnv()
	debMachine(m, "deb")
	units := loadOne(t, e, debSpecText(""), nil)
	if err := applyOnce(t, units[0].Step); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(m.Calls, "download https://x/1.2/a_1.2_amd64.deb") || !slices.Contains(m.Calls, "run "+debInstall) {
		t.Errorf("calls = %q, want the download and the install of %s", m.Calls, debFile)
	}
	m.Packages["a"] = effecttest.Package{Version: "1.2", Status: "ii"}
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("plan with a 1.2 installed = %q, want none", got)
	}
}

func TestDebRefusesAWrongSHA256(t *testing.T) {
	e, m, _, _ := testEnv()
	debMachine(m, "not the pinned file")
	err := applyOnce(t, loadOne(t, e, debSpecText(""), nil)[0].Step)
	if err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Errorf("apply = %v, want a wrong SHA-256", err)
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, "dpkg-deb") || strings.Contains(c, "apt-get") }) {
		t.Errorf("calls = %q, want dpkg never to see the file", m.Calls)
	}
}

func TestDebStopKeepsTheServiceStopped(t *testing.T) {
	e, m, u, _ := testEnv()
	debMachine(m, "deb")
	units := loadOne(t, e, debSpecText("stop = \"a.service\"\n"), nil)
	if err := applyOnce(t, units[0].Step); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(u.Calls, []string{"stop a.service", "disable a.service"}) {
		t.Errorf("calls = %q, want a.service stopped and disabled", u.Calls)
	}
	m.Packages["a"] = effecttest.Package{Version: "1.2", Status: "ii"}
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("plan with a installed = %q, want none: the stop goes with the install alone", got)
	}
}

func TestDebRefusesFieldsNoPlanTakes(t *testing.T) {
	e, _, _, _ := testEnv()
	for spec, want := range map[string]string{
		"[deb.a]\npackage = \"a\"\nversion = \"1\"\nurl = \"https://u\"\n":                             "a sha256 for each",
		"[deb.a]\npackage = \"a\"\nversion = \"1\"\nurl = \"https://u\"\nsha256 = { amd64 = \"x\" }\n": "64 lowercase hex",
		strings.Replace(debSpecText(""), "https://", "http://", 1):                                     "is no https URL",
		debSpecText("stop = \"a\"\n"):                                                                  "type's suffix",
	} {
		if _, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": spec})); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("load of\n%s= %v, want %q", spec, err, want)
		}
	}
}
