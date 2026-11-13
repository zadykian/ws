package host

import (
	"context"
	"io/fs"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

func TestHostKeyAdd(t *testing.T) {
	h, m := newHost(map[string]string{"WS_HOST_KEY": "yes"})
	m.Files[hostsOf] = []byte("a b")
	if err := h.Apply(context.Background(), step.Plan{}); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files[hostsOf]); got != "a b\nws "+hostLine+"\n" {
		t.Errorf("known_hosts = %q", got)
	}
}

func TestHostKeyCheck(t *testing.T) {
	h, m := newHost(nil)
	h.Check = true
	if _, err := h.Plan(context.Background()); err != nil {
		t.Fatal(err)
	}
	out := strings.Join(h.Out.(*effecttest.Out).Lines, "\n")
	if _, wrote := m.Files[hostsOf]; wrote || !strings.Contains(out, "a run would ask whether ws's console shows it") ||
		!strings.Contains(out, "diff: "+hostsOf) {
		t.Errorf("--check wrote %v and said %q", wrote, out)
	}
}

func TestHostKeyLink(t *testing.T) {
	for _, check := range []bool{false, true} {
		h, m := newHost(map[string]string{"WS_HOST_KEY": "yes"})
		h.Check = check
		m.Links[hostsOf] = "/d/hosts"
		m.Modes[hostsOf] = fs.ModeSymlink
		err := h.add(context.Background())
		said := strings.Join(h.Out.(*effecttest.Out).Lines, "\n")
		if err == nil || strings.Contains(said, "raw: ws "+hostLine) == check {
			t.Errorf("check %v: add = %v, said %q; want a stop, and the line unless checking", check, err, said)
		}
	}
}
