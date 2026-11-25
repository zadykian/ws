package gitcfg_test

import (
	"testing"

	"github.com/zadykian/ws/internal/gitcfg"
)

func TestXDGFileFollowsXDGConfigHomeOrTheHome(t *testing.T) {
	if got, want := gitcfg.XDGFile("", "/home/ws"), "/home/ws/.config/git/config"; got != want {
		t.Errorf("XDGFile unset = %q, want %q", got, want)
	}
	if got, want := gitcfg.XDGFile("/srv/cfg", "/home/ws"), "/srv/cfg/git/config"; got != want {
		t.Errorf("XDGFile set = %q, want %q", got, want)
	}
}

func TestXDGNoteNamesTheFileAndTheWinner(t *testing.T) {
	want := "/h/.config/git/config's settings of user.email come before the include in ~/.gitconfig, " +
		"so /etc/ws/gitconfig's win over them"
	if got := gitcfg.XDGNote("/h/.config/git/config", "user.email", gitcfg.Path); got != want {
		t.Errorf("XDGNote = %q, want %q", got, want)
	}
}
