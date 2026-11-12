package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/internal/sshcfg"
)

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

// mac.conf can name a copy before ws mac apply makes its directory. Apply makes the directory, and
// sets ~/.config/ws's mode, before the copy, so that a second plan finds nothing.
func TestMacKeyApplyMakesTheCopiesDirectory(t *testing.T) {
	k, m, _ := newMacKey(t, "WS_IDENTITY_PUB_KEY="+listed(p256)[0].Pub()+"\n")
	agent(m, p256)
	delete(m.Dirs, "/h/"+sshcfg.KeyCopies)
	m.Dirs["/h/.config/ws"] = 0o755
	p, err := k.Plan(context.Background())
	if err != nil || len(p.Changes) != 3 {
		t.Fatalf("Plan = %v, %v; want the mode, the directory and the copy", p.Changes, err)
	}
	if err := k.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if m.Dirs["/h/.config/ws"] != 0o700 || m.Dirs["/h/"+sshcfg.KeyCopies] != 0o755 || string(m.Files[copyOf(p256)]) != p256 {
		t.Errorf("Apply left %o, %o and %q", m.Dirs["/h/.config/ws"], m.Dirs["/h/"+sshcfg.KeyCopies], m.Files[copyOf(p256)])
	}
	if got := planned(k); got != "[] <nil>" {
		t.Errorf("Plan after Apply = %q", got)
	}
}
