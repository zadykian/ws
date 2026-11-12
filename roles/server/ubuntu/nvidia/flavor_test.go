package nvidia

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestPackageNamesFollowTheInstalledFlavors(t *testing.T) {
	for _, tc := range []struct {
		name     string
		packages effecttest.Packages
		want     []string
	}{
		{"generic", effecttest.Packages{"linux-image-generic": {Version: "7.0.0-42.42", Status: "ii"}},
			[]string{modulesPrefix + "generic=7.0.0-42.42"}},
		{"hwe and a removed generic", effecttest.Packages{"linux-image-generic": {Version: "7.0.0-14.14", Status: "rc"},
			"linux-image-generic-hwe-26.04": {Version: "7.0.0-43.43", Status: "ii"}},
			[]string{modulesPrefix + "generic-hwe-26.04=7.0.0-43.43"}},
		{"two", effecttest.Packages{"linux-image-generic": {Version: "7.0.0-42.42", Status: "ii"},
			"linux-image-generic-7.0": {Version: "7.0.0-43.43", Status: "ii"}},
			[]string{modulesPrefix + "generic=7.0.0-42.42", modulesPrefix + "generic-7.0=7.0.0-43.43"}},
	} {
		m := effecttest.NewMachine()
		m.Packages = tc.packages
		got, err := packageNames(t.Context(), m)
		if want := append(slices.Clone(driverPackages), tc.want...); err != nil || !slices.Equal(got, want) {
			t.Errorf("%s: packageNames = %q, %v; want %q", tc.name, got, err, want)
		}
	}
}

func TestPackageNamesRefuseAnotherKernel(t *testing.T) {
	m := effecttest.NewMachine()
	m.Packages = effecttest.Packages{"linux-image-generic-hwe-24.04": {Version: "6.8.0-90.91", Status: "ii"}}
	if _, err := packageNames(t.Context(), m); err == nil || !strings.Contains(err.Error(), "no NVIDIA modules") {
		t.Errorf("packageNames = %v; want the flavors refused", err)
	}
}
