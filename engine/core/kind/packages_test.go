package kind

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

const (
	aptUpdate  = "env DEBIAN_FRONTEND=noninteractive apt-get update"
	aptInstall = "env DEBIAN_FRONTEND=noninteractive apt-get -o DPkg::Lock::Timeout=600 " +
		"-o Dpkg::Options::=--force-confdef -o Dpkg::Options::=--force-confold install -y "
	packagesUnit = "[packages.a]\ntemplate = \"a.tmpl\"\n"
)

// loadPackages loads packages/a, whose template is tmpl, on m.
func loadPackages(t *testing.T, m *effecttest.Machine, tmpl string) []Unit {
	return loadOne(t, Env{M: m, Out: &effecttest.Out{}}, packagesUnit, map[string]string{"a.tmpl": tmpl})
}

func TestPackagesInstallsTheNamesOfItsTemplate(t *testing.T) {
	m := effecttest.NewMachine()
	m.Packages = effecttest.Packages{"aa": {Version: "1", Status: "ii"}}
	m.Results[aptUpdate] = effecttest.Result{}
	m.Results[aptInstall+"bb"] = effecttest.Result{}
	units := loadPackages(t, m, "aa\n  bb\n")
	if got := headlines(t, units[0].Step); !slices.Equal(got, []string{"install bb"}) {
		t.Fatalf("plan = %q, want install bb", got)
	}
	if err := applyOnce(t, units[0].Step); err != nil {
		t.Fatal(err)
	}
	if i := slices.Index(m.Calls, "run "+aptUpdate); i < 0 || !slices.Contains(m.Calls[i:], "run "+aptInstall+"bb") {
		t.Errorf("calls = %q, want apt-get update, then the install of bb with recommends", m.Calls)
	}
}

func TestPackagesFollowAnInstalledVersion(t *testing.T) {
	m := effecttest.NewMachine()
	m.Packages = effecttest.Packages{"linux-image-generic": {Version: "7.0.0-10.10", Status: "ii"}}
	units := loadPackages(t, m, `{{with installed_version "linux-image-generic"}}m-generic={{.}}{{end}}
{{with installed_version "linux-image-other"}}m-other={{.}}{{end}}
`)
	if got := headlines(t, units[0].Step); !slices.Equal(got, []string{"install m-generic=7.0.0-10.10"}) {
		t.Errorf("plan = %q, want the generic flavor's modules at its version alone", got)
	}
	m.Packages["m-generic"] = effecttest.Package{Version: "7.0.0-10.10", Status: "ii"}
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("plan with them installed = %q, want none", got)
	}
}

func TestPackagesRefusesABadName(t *testing.T) {
	for _, bad := range []string{"-o x", "a/b", "A", "a=", "a=1 2;b", "foo-", "foo=1-"} {
		m := effecttest.NewMachine()
		m.Packages = effecttest.Packages{}
		units := loadPackages(t, m, "aa "+bad+"\n")
		if _, err := units[0].Step.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "no package's name") {
			t.Errorf("plan of %q = %v, want a refusal", bad, err)
		}
	}
}

func TestPackagesNamingNoneIsInSync(t *testing.T) {
	m := effecttest.NewMachine()
	units := loadPackages(t, m, "{{if false}}aa{{end}}\n")
	if got := headlines(t, units[0].Step); got != nil || len(m.Calls) != 0 {
		t.Errorf("plan = %q, calls %q; want none, and no dpkg-query", got, m.Calls)
	}
}

func TestPackagesRefusesFieldsNoPlanTakes(t *testing.T) {
	e, _, _, _ := testEnv()
	for spec, want := range map[string]string{
		"[packages.a]\n":                      "packages take a template",
		"[packages.a]\ntemplate = \"gone\"\n": "gone",
	} {
		if _, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": spec})); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("load of\n%s= %v, want %q", spec, err, want)
		}
	}
}

// simulatedUnit is packages/a, whose names are only simulated.
const simulatedUnit = packagesUnit + "simulate = true\n"

func TestPackagesSimulateChecksEachName(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["apt-get install -s a1 b2"] = effecttest.Result{}
	units := loadOne(t, Env{M: m, Out: &effecttest.Out{}}, simulatedUnit, map[string]string{"a.tmpl": "a1\nb2\n"})
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("plan = %q, want none", got)
	}
	if !slices.Equal(m.Calls, []string{"run apt-get install -s a1 b2"}) {
		t.Errorf("calls = %q, want the simulation alone: no dpkg-query, update or install", m.Calls)
	}
}

func TestPackagesSimulateFailsOnAMissingName(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["apt-get install -s a1"] = effecttest.Result{Err: errors.New("E: Unable to locate package a1")}
	units := loadOne(t, Env{M: m, Out: &effecttest.Out{}}, simulatedUnit, map[string]string{"a.tmpl": "a1\n"})
	if _, err := units[0].Step.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "Unable to locate package a1") {
		t.Errorf("plan = %v, want apt's error", err)
	}
}

func TestPackagesSimulateRefusesABadName(t *testing.T) {
	m := effecttest.NewMachine()
	units := loadOne(t, Env{M: m, Out: &effecttest.Out{}}, simulatedUnit, map[string]string{"a.tmpl": "a1 -o x\n"})
	if _, err := units[0].Step.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "no package's name") || len(m.Calls) != 0 {
		t.Errorf("plan = %v, calls %q; want a refusal before apt-get runs", err, m.Calls)
	}
}
