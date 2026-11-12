package nvidia

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
)

func TestStepsPlanNothingWithoutTheGPU(t *testing.T) {
	m, out := withDevice("0x030000", "0x1af4"), &effecttest.Out{}
	m.Results[checkLine] = effecttest.Result{}
	var names []string
	for _, s := range Steps(values.Default(), secrets.Secrets{}, m, out) {
		names = append(names, s.Name())
		if p, err := s.Plan(t.Context()); err != nil || len(p.Changes) != 0 {
			t.Errorf("%s: Plan = %v, %v; want none", s.Name(), p, err)
		}
	}
	if want := []string{gpuStep, packagesStep, driverStep}; !slices.Equal(names, want) {
		t.Errorf("steps = %q, want %q", names, want)
	}
}

func TestPackagesInstallTheFlavorsModules(t *testing.T) {
	m := withDevice("0x030000", "0x10de")
	m.Packages = effecttest.Packages{
		"linux-image-generic":     {Version: "7.0.0-42.42", Status: "ii"},
		modulesPrefix + "generic": {Version: "7.0.0-14.14", Status: "ii"},
	}
	want := append(slices.Clone(driverPackages), modulesPrefix+"generic=7.0.0-42.42")
	p, err := Steps(values.Default(), secrets.Secrets{}, m, &effecttest.Out{})[1].Plan(t.Context())
	if err != nil || len(p.Changes) != 1 || p.Changes[0].String() != "install "+strings.Join(want, " ") {
		t.Errorf("Plan = %v, %v; want an install of %q", p, err, want)
	}
}
