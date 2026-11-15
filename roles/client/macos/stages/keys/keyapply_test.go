package keys

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/roles/client/macos/values"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

// With no terminal, Apply stops at ws's question, before git's.
func TestMacKeyApplyStopsAtTheQuestion(t *testing.T) {
	k, m, _ := newMacKey(t, "")
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

// values.yml can name a copy before ws client macos apply makes its directory. Apply makes the
// directory before the copy, so that a second plan finds nothing.
func TestMacKeyApplyMakesTheCopiesDirectory(t *testing.T) {
	k, m, _ := newMacKey(t, gitKey+values.Text("WS_IDENTITY_PUB_KEY", listed(p256)[0].Pub()))
	agent(m, p256)
	delete(m.Dirs, sshcfg.KeyCopies)
	p, err := k.Plan(context.Background())
	if err != nil || len(p.Changes) != 2 {
		t.Fatalf("Plan = %v, %v; want the directory and the copy", p.Changes, err)
	}
	if err := k.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if m.Dirs[sshcfg.KeyCopies] != 0o755 || string(m.Files[copyOf(p256)]) != p256 {
		t.Errorf("Apply left %o and %q", m.Dirs[sshcfg.KeyCopies], m.Files[copyOf(p256)])
	}
	if got := planned(k); got != "[] <nil>" {
		t.Errorf("Plan after Apply = %q", got)
	}
}
