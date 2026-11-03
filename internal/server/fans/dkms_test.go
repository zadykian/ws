package fans

import (
	"slices"
	"testing"
)

func TestParseDKMS(t *testing.T) {
	const v, k = "7e3d7b36c55f", "7.0.0-38-generic"
	for _, tc := range []struct {
		name, out string
		versions  []string
		installed bool
	}{
		{"nothing", "", nil, false},
		{"installed", "nct6687d/" + v + ", " + k + ", x86_64: installed\n", []string{v}, true},
		{"added only", "nct6687d/" + v + ": added\n", []string{v}, false},
		{"another kernel", "nct6687d/" + v + ", 6.8.0-1-generic, x86_64: installed\n", []string{v}, false},
		{"built, not installed", "nct6687d/" + v + ", " + k + ", x86_64: built\n", []string{v}, false},
		{"two versions once each", "nct6687d/old, " + k + ", x86_64: installed\nnct6687d/old, 6.8.0, x86_64: installed\nnct6687d/" + v + ": added\n", []string{"old", v}, false},
		{"a longer version is not the pin", "nct6687d/" + v + "x, " + k + ", x86_64: installed\n", []string{v + "x"}, false},
	} {
		got := parseDKMS(tc.out, v, k)
		if !slices.Equal(got.versions, tc.versions) || got.installed != tc.installed {
			t.Errorf("%s: parseDKMS = %+v, want versions %q installed %v", tc.name, got, tc.versions, tc.installed)
		}
	}
}
