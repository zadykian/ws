package fresh

import "testing"

func TestTheURLIsTheReleasesDeb(t *testing.T) {
	want := "https://github.com/sinelaw/fresh/releases/download/v0.5.7/fresh-editor_0.5.7-1_arm64.deb"
	if got := debURL("arm64"); got != want {
		t.Errorf("got %s", got)
	}
}
