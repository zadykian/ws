package wizard

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestMacKeyMadeInstallsSecretive(t *testing.T) {
	k, m, _ := newMacKey(t, "")
	agent(m)
	m.Results["brew install --cask secretive"] = effecttest.Result{}
	m.Results["open -a Secretive"] = effecttest.Result{}
	k.Sleep = func(time.Duration) { agent(m, p256) }
	got, err := k.made(context.Background(), reason, nil)
	if err != nil || got.Line+"\n" != p256 {
		t.Fatalf("made = %q, %v", got.Line, err)
	}
	want := []string{"run brew install --cask secretive", "run open -a Secretive"}
	if calls := append(callsWith(m, "brew"), callsWith(m, "open")...); !slices.Equal(calls, want) {
		t.Errorf("calls = %q, want %q", calls, want)
	}
}

func TestMacKeyMadeStopsWhereTheCaskFails(t *testing.T) {
	k, m, _ := newMacKey(t, "")
	m.Results["brew install --cask secretive"] = effecttest.Result{Err: errors.New("exit status 1")}
	if _, err := k.made(context.Background(), reason, nil); err == nil ||
		err.Error() != "Mac key: brew install --cask secretive failed: install Secretive, then run the wizard again" {
		t.Errorf("made = %v", err)
	}
}

// Secretive's agent was not there at the question, and lists a key kept from an earlier install
// once it starts: no answer named it, so made stops.
func TestMacKeyMadeStopsAtAKeyNotOffered(t *testing.T) {
	k, m, _ := newMacKey(t, "")
	m.Dirs["/h/Applications/Secretive.app"] = 0o755
	m.Results["open -a Secretive"] = effecttest.Result{}
	k.Sleep = func(time.Duration) { agent(m, p256) }
	want := "Mac key: Secretive's agent lists " + listed(p256)[0].Label() + ", which the question did not offer: " +
		"run the wizard again to pick it, or to make a new one"
	if _, err := k.made(context.Background(), reason, nil); err == nil || err.Error() != want {
		t.Errorf("made = %v, want %q", err, want)
	}
}
