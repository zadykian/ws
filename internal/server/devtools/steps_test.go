package devtools

import (
	"context"
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
)

func TestStepsOrderAndWaits(t *testing.T) {
	steps := Steps(values.Default(), secrets.Secrets{}, effecttest.NewMachine(), &effecttest.Units{}, &effecttest.Out{})
	var names, waits []string
	for _, s := range steps {
		names, waits = append(names, s.Name()), append(waits, s.WaitsOn())
	}
	if want := []string{"Devtools repositories", "Devtools packages", "Devtools gh telemetry", "Devtools gh login", "Devtools Go", "Devtools gopls", "Devtools Delve", "Devtools roslyn-language-server", "Devtools postgres-language-server", "Devtools docker-language-server", "Devtools npm servers", "Devtools rustup"}; !slices.Equal(names, want) {
		t.Errorf("steps %q, want %q", names, want)
	}
	if want := []string{"", "Devtools repositories", "Devtools packages", "Devtools packages", "", "Devtools Go", "Devtools Go", "Devtools packages", "", "", "Devtools packages", ""}; !slices.Equal(waits, want) {
		t.Errorf("waits %q, want %q", waits, want)
	}
}

func TestEachStepWaitsOnAnEarlierOne(t *testing.T) {
	steps := Steps(values.Default(), secrets.Secrets{}, effecttest.NewMachine(), &effecttest.Units{}, &effecttest.Out{})
	for i, s := range steps {
		if w := s.WaitsOn(); w != "" && !slices.ContainsFunc(steps[:i], func(e step.Step) bool { return e.Name() == w }) {
			t.Errorf("%q waits on %q, which is not an earlier step", s.Name(), w)
		}
	}
}

func TestPackageNames(t *testing.T) {
	want := []string{"nodejs=24.*", "gh", "glow", "dotnet-sdk-8.0", "dotnet-sdk-10.0"}
	if got := packageNames(); !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPackagesPlanChromiumLibrariesWithoutRecommends(t *testing.T) {
	v := values.Default()
	v.Devtools.ChromiumLibraries = []string{"libnss3"}
	m := effecttest.NewMachine()
	m.Packages = effecttest.Packages{"nodejs": {Version: "24.1.0", Status: "ii"}, "gh": {Version: "2", Status: "ii"},
		"glow": {Version: "2", Status: "ii"}, "dotnet-sdk-8.0": {Version: "8.0", Status: "ii"},
		"dotnet-sdk-10.0": {Version: "10.0", Status: "ii"}}
	plan, err := Steps(v, secrets.Secrets{}, m, &effecttest.Units{}, &effecttest.Out{})[1].Plan(context.Background())
	if err != nil || len(plan.Changes) != 1 || plan.Changes[0].String() != "install libnss3" {
		t.Errorf("got %v, %v", plan.Changes, err)
	}
}
