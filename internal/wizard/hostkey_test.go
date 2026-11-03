package wizard

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

const (
	hostLine = "ssh-ed25519 AAAAC3"
	hostFP   = "SHA256:abc"
	hostsOf  = "/h/.ssh/known_hosts"
)

// newHost returns the stage on a machine whose ws answers at 10.1.1.2:22 with hostLine, and whose
// known_hosts lacks ws.
func newHost(env map[string]string) (*HostKey, *effecttest.Machine) {
	m := effecttest.NewMachine()
	m.Files["/h/mac.conf"] = []byte("WS_TARGET=lan\nWS_REMOTE=ws.example.org\n")
	m.Dirs["/l"] = 0o700
	m.Results["ssh -G ws-ssh"] = effecttest.Result{Out: []byte("user root\nhostname 10.1.1.2\nport 22\n" +
		"userknownhostsfile ~/.ssh/known_hosts\n")}
	m.Results["ssh-keygen -F ws -f "+hostsOf] = effecttest.Result{Err: errors.New("none")}
	m.Results["ssh-keyscan -T 10 -t ed25519 -p 22 10.1.1.2"] = effecttest.Result{
		Out: []byte("10.1.1.2 " + hostLine + "\n")}
	m.Results["ssh-keygen -l -f /l/host.pub"] = effecttest.Result{Out: []byte("256 " + hostFP + " ws (ED25519)\n")}
	return &HostKey{Conf: "/h/mac.conf", Lock: "/l", Tty: true, M: m, Out: &effecttest.Out{},
		SSH: SSH{M: m, Lock: "/l", Home: "/h"},
		Env: func(k string) (string, bool) { v, ok := env[k]; return v, ok }}, m
}

func TestHostKeyPlan(t *testing.T) {
	h, m := newHost(nil)
	if p, err := h.Plan(context.Background()); err != nil || len(p.Changes) != 1 {
		t.Errorf("Plan = %v, %v; want one change", p, err)
	}
	m.Results["ssh-keygen -F ws -f "+hostsOf] = effecttest.Result{}
	if p, err := h.Plan(context.Background()); err != nil || len(p.Changes) != 0 {
		t.Errorf("Plan of a known key = %v, %v; want none", p, err)
	}
	m.Results["ssh -G ws-ssh"] = effecttest.Result{Out: []byte("hostname ws-ssh\n")}
	_, err := h.Plan(context.Background())
	if err == nil || !strings.Contains(err.Error(), "and no WS_REMOTE_PORT; set those in mac.conf as docs/router.md has them") {
		t.Errorf("Plan with no way: %v", err)
	}
}
