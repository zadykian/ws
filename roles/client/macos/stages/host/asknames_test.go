package host

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/remote"
)

func TestHostKeyAsksNamingTheServer(t *testing.T) {
	h, m := labHost(nil)
	m.Results["ssh-keyscan -T 10 -t ed25519 -p 22 10.1.1.2"] = effecttest.Result{}
	h.Check = true
	if err := h.add(context.Background()); err != nil || !strings.Contains(strings.Join(h.Out.(*effecttest.Out).Lines, "\n"),
		"lab doesn't answer at 10.1.1.2:22 with an ed25519 key; a run would ask its address now") {
		t.Errorf("--check = %v, said %q", err, h.Out.(*effecttest.Out).Lines)
	}
	h.Check = false
	h.P = &effecttest.Prompt{T: t, Answers: map[string]string{"lab's address now, as the router's leases show it": ""}}
	w := remote.Way{Host: "10.1.1.2", Port: "22"}
	_, _, err := h.question().ask(context.Background(), "lan", w)
	said := strings.Join(h.Out.(*effecttest.Out).Lines, "\n")
	if err == nil || !strings.Contains(err.Error(), "no address for lab; once") || !strings.Contains(said, "isn't installed on lab yet") {
		t.Errorf("ask = %v, said %q; want lab", err, said)
	}
	h.P = &effecttest.Prompt{T: t, Answers: map[string]string{"lab's address now, as ip -4 address prints it at lab's console": ""}}
	if _, _, err = h.question().ask(context.Background(), "mac-vm", w); err == nil || !strings.Contains(err.Error(), "no address for lab") {
		t.Errorf("ask on a VM = %v; want lab's console in the question", err)
	}
	h.Tty = false
	if _, _, err = h.question().ask(context.Background(), "lan", w); err == nil || !strings.Contains(err.Error(), "lab doesn't answer at 10.1.1.2:22") {
		t.Errorf("ask with no terminal = %v; want lab", err)
	}
}
