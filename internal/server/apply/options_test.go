package apply

import (
	"slices"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want Options
		err  string
	}{
		{"none", nil, Options{}, ""},
		{"check", []string{"--check"}, Options{Check: true}, ""},
		{"only", []string{"--only", "base,fans"}, Options{Only: []string{"base", "fans"}}, ""},
		{"only twice", []string{"--only=base", "--only", "fans"}, Options{Only: []string{"base", "fans"}}, ""},
		{"playbook args", []string{"--check", "--", "-e", "@f", "--skip-tags", "systemd"},
			Options{Check: true, Args: []string{"-e", "@f", "--skip-tags", "systemd"}}, ""},
		{"bare dashes", []string{"--"}, Options{Args: []string{}}, ""},
		{"only needs names", []string{"--only"}, Options{}, "needs role names"},
		{"empty name", []string{"--only", "base,"}, Options{}, "empty role name"},
		{"unknown", []string{"--tags", "base"}, Options{}, `unknown argument "--tags"`},
		{"-C after --", []string{"--", "-C"}, Options{}, "use ws server apply --check"},
		{"--check after --", []string{"--", "--check"}, Options{}, "use ws server apply --check"},
		{"-t after --", []string{"--", "-t", "base"}, Options{}, "--only"},
		{"-tX after --", []string{"--", "-tbase"}, Options{}, "--only"},
		{"--tags after --", []string{"--", "--tags", "base"}, Options{}, "--only"},
		{"--tags= after --", []string{"--", "--tags=base"}, Options{}, "--only"},
		{"-e of a path with t", []string{"--", "-e@/tmp/x.yml"}, Options{Args: []string{"-e@/tmp/x.yml"}}, ""},
	} {
		got, err := Parse(tc.args)
		if tc.err != "" {
			if err == nil || !strings.Contains(err.Error(), tc.err) {
				t.Errorf("%s: err = %v, want it to hold %q", tc.name, err, tc.err)
			}
			continue
		}
		if err != nil || got.Check != tc.want.Check || !slices.Equal(got.Only, tc.want.Only) || !slices.Equal(got.Args, tc.want.Args) {
			t.Errorf("%s: Parse = %+v, %v, want %+v", tc.name, got, err, tc.want)
		}
	}
}
