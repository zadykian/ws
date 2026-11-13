package wizard

import (
	"context"
	"github.com/zadykian/ws/engine/core/remote"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestHostKeyAsksFirstAddress(t *testing.T) {
	h, m := newHost(map[string]string{"WS_HOST_KEY": "yes"})
	h.P = &effecttest.Prompt{T: t, Answers: map[string]string{"ws's address now, as the router's leases show it": "10.1.1.9"}}
	m.Results["ssh-keyscan -T 10 -t ed25519 -p 22 10.1.1.2"] = effecttest.Result{}
	m.Files["/l/"+remote.FirstFile] = []byte("10.1.1.9\n")
	m.Results[wayCall(h.SSH)] = effecttest.Result{Out: []byte(
		"hostname 10.1.1.9\nport 22\nuserknownhostsfile ~/.ssh/known_hosts\n")}
	delete(m.Files, "/l/"+remote.FirstFile)
	m.Results["ssh-keyscan -T 10 -t ed25519 -p 22 10.1.1.9"] = effecttest.Result{Out: []byte("x " + hostLine + "\n")}
	w, line, err := h.question().ask(context.Background(), "lan", remote.Way{Host: "10.1.1.2", Port: "22"})
	if err != nil || w.Host != "10.1.1.9" || line != hostLine {
		t.Fatalf("ask = %+v, %q, %v; want the key at 10.1.1.9", w, line, err)
	}
	if got := string(m.Files["/l/"+remote.FirstFile]); got != "10.1.1.9\n" {
		t.Errorf("first address file = %q", got)
	}
}
