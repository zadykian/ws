package wizard

import (
	"strings"
	"testing"
)

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

func TestMacKeyQuestionWithNoTerminal(t *testing.T) {
	want := "Mac key: no terminal to ask for git's signing key: add one of these lines to mac.conf, " +
		"WS_GIT_SIGNING_PUB_KEY=" + listed(p256)[0].Pub() + " for " + p256Label + "; or run the wizard at a terminal"
	if err := noTerminal(gitUse, listed(p256, mldsa)); err == nil || err.Error() != want {
		t.Errorf("noTerminal = %v, want %q", err, want)
	}
	if err := noTerminal(gitUse, listed(mldsa)); err == nil || !strings.Contains(err.Error(), "lists no P-256 key") {
		t.Errorf("noTerminal of no P-256 key = %v", err)
	}
}
