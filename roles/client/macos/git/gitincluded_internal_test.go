package git

import (
	"strings"
	"testing"
)

const dir = "/h/.config/ws"

func TestIncludedFindsTheIncludeFirstAndOnce(t *testing.T) {
	for name, tc := range map[string]struct {
		paths []string
		first string
		want  bool
		err   string
	}{
		"none":  {nil, "user.name", false, ""},
		"other": {[]string{"/x/y"}, "include.path", false, ""},
		"first": {[]string{dir + "/gitconfig", "/x"}, "include.path", true, ""},
		"below": {[]string{dir + "/gitconfig"}, "user.name", false, "below its own settings"},
		"twice": {[]string{dir + "/gitconfig", dir + "/gitconfig"}, "include.path", false, "2 times"},
	} {
		got, err := included(tc.paths, tc.first, dir)
		if got != tc.want || (err == nil) != (tc.err == "") || (err != nil && !strings.Contains(err.Error(), tc.err)) {
			t.Errorf("%s: Included = %v, %v; want %v, %q", name, got, err, tc.want, tc.err)
		}
	}
}

func TestOwnListsEachSettingOnce(t *testing.T) {
	names := []string{"core.editor", "user.name", "gpg.format", "user.name", "user.signingkey", "gpg.ssh.program"}
	if got, want := ownSettings(names), "user.name, gpg.format, user.signingkey, gpg.ssh.program"; got != want {
		t.Errorf("Own = %q, want %q", got, want)
	}
	if got := ownSettings(nil); got != "" {
		t.Errorf("Own(nil) = %q", got)
	}
}
