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
		{"playbook args", []string{"--check", "--", "-e", "a=b", "--skip-tags", "systemd"},
			Options{Check: true, Args: []string{"-e", "a=b", "--skip-tags", "systemd"}}, ""},
		{"secrets", []string{"--secrets", "/f.age"}, Options{Secrets: "/f.age"}, ""},
		{"secrets=", []string{"--secrets=/f.age", "--check"}, Options{Secrets: "/f.age", Check: true}, ""},
		{"secrets needs a file", []string{"--secrets"}, Options{}, "needs a file"},
		{"secrets twice", []string{"--secrets=a", "--secrets=b"}, Options{}, "one file"},
		{"-e @file after --", []string{"--", "-e", "@/v.yml"}, Options{}, "--secrets"},
		{"-e@file after --", []string{"--", "-e@/v.yml"}, Options{}, "--secrets"},
		{"--extra-vars=@ after --", []string{"--", "--extra-vars=@/v.yml"}, Options{}, "--secrets"},
		{"--ask-vault-pass after --", []string{"--", "--ask-vault-pass"}, Options{}, "--secrets"},
		{"--vault-password-file= after --", []string{"--", "--vault-password-file=/dev/fd/3"}, Options{}, "--secrets"},
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
		{"-e of a variable with t", []string{"--", "-etx=1"}, Options{Args: []string{"-etx=1"}}, ""},
	} {
		got, err := Parse(tc.args)
		if tc.err != "" {
			if err == nil || !strings.Contains(err.Error(), tc.err) {
				t.Errorf("%s: err = %v, want it to hold %q", tc.name, err, tc.err)
			}
			continue
		}
		if err != nil || got.Check != tc.want.Check || got.Secrets != tc.want.Secrets || !slices.Equal(got.Only, tc.want.Only) || !slices.Equal(got.Args, tc.want.Args) {
			t.Errorf("%s: Parse = %+v, %v, want %+v", tc.name, got, err, tc.want)
		}
	}
}

func TestExtraVars(t *testing.T) {
	args := []string{"-e", "a=1 b=2", "--tags-not", "-ec=3", "--extra-vars=d=4", "--extra-vars", "e=5", "-e", "@f", "-e", "{\"g\": 6}"}
	want := []string{"a=1", "b=2", "c=3", "d=4", "e=5"}
	if got := ExtraVars(args); !slices.Equal(got, want) {
		t.Errorf("ExtraVars = %q, want %q", got, want)
	}
}
