package nvidia

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

// flavorQuery is the command line of flavorLine.
var flavorQuery = "dpkg-query " + strings.Join(flavorLine, " ")

func TestPackageNamesFollowTheInstalledFlavors(t *testing.T) {
	for _, tc := range []struct {
		name, out string
		want      []string
	}{
		{"generic", "linux-image-generic 7.0.0-42.42 ii \n", []string{modulesPrefix + "generic=7.0.0-42.42"}},
		{"hwe and a removed generic", "linux-image-generic 7.0.0-14.14 rc \nlinux-image-generic-hwe-26.04 7.0.0-43.43 ii \n",
			[]string{modulesPrefix + "generic-hwe-26.04=7.0.0-43.43"}},
		{"two", "linux-image-generic 7.0.0-42.42 ii \nlinux-image-generic-7.0 7.0.0-43.43 ii \n",
			[]string{modulesPrefix + "generic=7.0.0-42.42", modulesPrefix + "generic-7.0=7.0.0-43.43"}},
	} {
		m := effecttest.NewMachine()
		m.Results[flavorQuery] = effecttest.Result{Out: []byte(tc.out)}
		got, err := packageNames(t.Context(), m)
		if want := append(slices.Clone(driverPackages), tc.want...); err != nil || !slices.Equal(got, want) {
			t.Errorf("%s: packageNames = %q, %v; want %q", tc.name, got, err, want)
		}
	}
}

func TestPackageNamesRefuseAnotherKernel(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[flavorQuery] = effecttest.Result{Out: []byte("linux-image-generic-hwe-24.04 6.8.0-90.91 ii \n")}
	if _, err := packageNames(t.Context(), m); err == nil || !strings.Contains(err.Error(), "no NVIDIA modules") {
		t.Errorf("packageNames = %v; want the flavors refused", err)
	}
}
