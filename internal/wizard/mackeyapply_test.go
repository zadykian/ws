package wizard

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/setup"
	"github.com/zadykian/ws/step"
)

func TestMacKeyApplyMakes(t *testing.T) {
	k, m, _ := newMacKey(t, "")
	m.Results["ssh-keygen -q -t ed25519 -a 100 -f /h/k"] = effecttest.Result{}
	m.Results["ssh-add --apple-use-keychain /h/k"] = effecttest.Result{Err: errors.New("no agent")}
	err := k.Apply(context.Background(), step.Plan{Changes: []step.Change{makeKey{"/h/k"}}})
	want := "Mac key: /h/k is made, but ssh-add cannot give it to the agent: run ssh-add --apple-use-keychain /h/k, " +
		"which asks its passphrase, then run the wizard again"
	if err == nil || err.Error() != want {
		t.Errorf("Apply = %v, want %q", err, want)
	}
}

func TestMacKeyApplyPins(t *testing.T) {
	k, m, _ := newMacKey(t, "WS_REMOTE=\n")
	m.Results["mkdir -p /h/.config/ws"] = effecttest.Result{}
	w := setup.Write{Path: confPathForTest, Before: []byte("WS_REMOTE=\n"),
		After: []byte("WS_REMOTE=\nWS_IDENTITY_PUB_KEY=~/.ssh/id_ed25519.pub\n"), Exists: true}
	if err := k.Apply(context.Background(), step.Plan{Changes: []step.Change{w}}); err != nil {
		t.Fatalf("Apply = %v", err)
	}
	if got := string(m.Files[confPathForTest]); got != string(w.After) {
		t.Errorf("mac.conf = %q", got)
	}
}

// With no terminal, Apply stops at ws's question, before git's.
func TestMacKeyApplyStopsAtTheQuestion(t *testing.T) {
	k, m, _ := newMacKey(t, gitConf)
	agent(m, p256)
	p, err := k.Plan(context.Background())
	if err != nil || len(p.Changes) != 2 {
		t.Fatalf("Plan = %v, %v; want both questions", p.Changes, err)
	}
	err = k.Apply(context.Background(), p)
	if err == nil || !strings.HasPrefix(err.Error(), "Mac key: no terminal to ask for ws's login key") {
		t.Errorf("Apply = %v, want the stop at ws's question", err)
	}
	if _, ok := m.Files[copyOf(p256)]; ok {
		t.Errorf("Apply wrote %s with no answer", copyOf(p256))
	}
}
