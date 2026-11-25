package gitcfg_test

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/internal/gitcfg"
)

func TestIncludedFindsTheIncludeFirstAndOnce(t *testing.T) {
	for name, tc := range map[string]struct {
		paths []string
		first string
		want  bool
		err   string
	}{
		"none":     {nil, "user.name", false, ""},
		"other":    {[]string{"/x/y"}, "include.path", false, ""},
		"first":    {[]string{gitcfg.Path, "/x"}, "include.path", true, ""},
		"below":    {[]string{gitcfg.Path}, "user.name", false, "below its own settings"},
		"twice":    {[]string{gitcfg.Path, gitcfg.Path}, "include.path", false, "2 times"},
		"relative": {[]string{".config/ws/gitconfig"}, "include.path", false, ""},
	} {
		got, err := gitcfg.Included(tc.paths, tc.first)
		if got != tc.want || (err == nil) != (tc.err == "") || (err != nil && !strings.Contains(err.Error(), tc.err)) {
			t.Errorf("%s: Included = %v, %v; want %v, %q", name, got, err, tc.want, tc.err)
		}
	}
}

func TestIncludeIsThePathAtTheTop(t *testing.T) {
	if want := "[include]\n\tpath = /etc/ws/gitconfig\n"; gitcfg.Include != want {
		t.Errorf("Include = %q, want %q", gitcfg.Include, want)
	}
}
