package wizard

import "testing"

// The labels of p256 and gitP256, by their comments and fingerprints.
const (
	p256Label = "me (SHA256:cJ6AyISHokEeHuTfufIqhhSS0gxHZRUMDHlKvXD4FHw)"
	gitLabel  = "me (SHA256:CcCKY/wrEaUM+I629sBixyeWSgyCiAj+RsdArzoziXo)"
)

func TestMacKeyQuestionString(t *testing.T) {
	want := "Mac key: would ask for git's signing key, in Secretive: " + p256Label + ", ws's login key; " +
		gitLabel + "; make a new one (the default)"
	q := keyQuestion(gitUse, listed(p256, mldsa, gitP256), listed(p256)[0].Name)
	if got := (askKey{gitUse, q}).String(); got != want {
		t.Errorf("String = %q, want %q", got, want)
	}
}
