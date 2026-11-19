package git

import (
	"errors"
	"strings"
	"testing"
)

func TestIncludedAtTheTop(t *testing.T) {
	_, sm, _ := gitFixture()
	sm.paths["[include]"] = "/etc/ws/gitconfig\n"
	sm.answers["[include]"] = "include.path\nuser.name\n"
	if holds, err := Included(t.Context(), sm, "~/.gitconfig", []byte(included+"[user]\n\tname = x\n")); !holds || err != nil {
		t.Errorf("Included = %v, %v; want it held", holds, err)
	}
}

func TestIncludedMissing(t *testing.T) {
	_, sm, _ := gitFixture()
	sm.answers["[user]"] = "user.name\n"
	if holds, err := Included(t.Context(), sm, "~/.gitconfig", []byte("[user]\n\tname = x\n")); holds || err != nil {
		t.Errorf("Included = %v, %v; want it lacking, with no error", holds, err)
	}
}

func TestIncludedRefusesBelowOrTwice(t *testing.T) {
	_, sm, _ := gitFixture()
	sm.answers["[user]"] = "user.name\ninclude.path\n"
	sm.paths["[user]"] = "/etc/ws/gitconfig\n"
	sm.answers["[include]"] = "include.path\ninclude.path\n"
	sm.paths["[include]"] = strings.Repeat("/etc/ws/gitconfig\n", 2)
	for text, want := range map[string]string{
		"[user]\n\tname = x\n" + included: "~/.gitconfig includes /etc/ws/gitconfig below its own settings",
		included + included:               "~/.gitconfig includes",
	} {
		holds, err := Included(t.Context(), sm, "~/.gitconfig", []byte(text))
		var out misplaced
		if holds || !errors.As(err, &out) || !strings.HasPrefix(err.Error(), want) {
			t.Errorf("%q: Included = %v, %v; want %q", text, holds, err, want)
		}
	}
}
