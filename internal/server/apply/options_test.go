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
		{"secrets", []string{"--secrets", "/f.age"}, Options{Secrets: "/f.age"}, ""},
		{"secrets=", []string{"--secrets=/f.age", "--check"}, Options{Secrets: "/f.age", Check: true}, ""},
		{"secrets needs a file", []string{"--secrets"}, Options{}, "needs a file"},
		{"secrets twice", []string{"--secrets=a", "--secrets=b"}, Options{}, "one file"},
		{"only needs names", []string{"--only"}, Options{}, "needs role names"},
		{"empty name", []string{"--only", "base,"}, Options{}, "empty role name"},
		{"unknown", []string{"--tags", "base"}, Options{}, `unknown argument "--tags"`},
		{"bare dashes", []string{"--"}, Options{}, `unknown argument "--"`},
		{"playbook argument", []string{"--", "--skip-tags", "systemd"}, Options{}, `unknown argument "--"`},
	} {
		got, err := Parse(tc.args)
		if tc.err != "" {
			if err == nil || !strings.Contains(err.Error(), tc.err) {
				t.Errorf("%s: err = %v, want it to hold %q", tc.name, err, tc.err)
			}
			continue
		}
		if err != nil || got.Check != tc.want.Check || got.Secrets != tc.want.Secrets || !slices.Equal(got.Only, tc.want.Only) {
			t.Errorf("%s: Parse = %+v, %v, want %+v", tc.name, got, err, tc.want)
		}
	}
}
