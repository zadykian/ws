package host

import (
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/remote"
)

// labHost is newHost for the server lab, whose lines must not name ws.
func labHost(env map[string]string) (*Key, *effecttest.Machine) {
	h, m := newHost(env)
	m.Results[wayCall(remote.SSH{Host: "lab", M: m, Lock: "/l", Home: "/h", Ways: MacWays{M: m, Conf: "/h/mac.conf"}})] = m.Results[wayCall(h.SSH)]
	h.SSH.Host = "lab"
	m.Results["ssh-keygen -F lab -f "+hostsOf] = effecttest.Result{Err: errors.New("none")}
	return h, m
}

func TestHostKeyPlanNamesTheServer(t *testing.T) {
	h, m := labHost(nil)
	p, err := h.Plan(context.Background())
	if err != nil || len(p.Changes) != 1 || !strings.Contains(p.Changes[0].String(), "would add lab's key to") {
		t.Errorf("Plan = %v, %v; want lab's key", p, err)
	}
	m.Files["/h/mac.conf"] = []byte("WS_TARGET=lan\n")
	if _, err = h.Plan(context.Background()); err == nil || !strings.Contains(err.Error(), "no direct way to lab from this network") {
		t.Errorf("Plan with no way = %v; want lab", err)
	}
}

func TestHostKeyAddNamesTheServer(t *testing.T) {
	h, m := labHost(map[string]string{"WS_HOST_KEY": "yes"})
	m.Results[wayCall(h.SSH)] = effecttest.Result{Out: []byte("user root\nhostname ws.example.org\nport 22\n" +
		"userknownhostsfile ~/.ssh/known_hosts\n")}
	m.Results["ssh-keyscan -T 10 -t ed25519 -p 22 ws.example.org"] = effecttest.Result{Out: []byte("x " + hostLine + "\n")}
	if err := h.add(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(h.Out.(*effecttest.Out).Lines, "\n"); !strings.Contains(got, "through WS_REMOTE, lab's new sshd faces the internet") {
		t.Errorf("said %q; want lab's sshd", got)
	}
	m.Links[hostsOf], m.Modes[hostsOf] = "/d/hosts", fs.ModeSymlink
	h.Check = true
	if err := h.write(hostsOf, hostLine); err == nil || !strings.Contains(err.Error(), "a run would print lab's line for its file") {
		t.Errorf("write over a link under --check = %v; want lab", err)
	}
}
