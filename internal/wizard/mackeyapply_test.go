package wizard

import (
	"context"
	"strings"
	"testing"
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
