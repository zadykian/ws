package wizard

import (
	"context"
	"testing"
	"time"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
)

const (
	pubText = "ssh-ed25519 AAAAC3 me@mac\n"
	lockDir = "/tmp/ws-wizard.lock"
)

func newMacKey(t *testing.T, conf string) (*MacKey, *effecttest.Machine, *effecttest.Out) {
	m := effecttest.NewMachine()
	m.Files[confPathForTest] = []byte(conf)
	out := &effecttest.Out{}
	return &MacKey{Conf: confPathForTest, Home: "/h", Lock: lockDir, M: m, P: &effecttest.Prompt{T: t}, Out: out,
		Sleep: func(time.Duration) {}}, m, out
}

func TestMacKeyListed(t *testing.T) {
	k, m, out := newMacKey(t, "")
	m.Files["/h/k.pub"] = []byte(pubText)
	m.Results["env SSH_AUTH_SOCK=/s ssh-add -L"] = effecttest.Result{Out: []byte("ssh-rsa BBBB x\nssh-ed25519 AAAAC3 other\n")}
	if !k.listed(context.Background(), "/h/k.pub", "/s") || k.listed(context.Background(), "/h/none.pub", "/s") {
		t.Error("listed did not find the key of the .pub, or found a missing one")
	}
	m.Results["env SSH_AUTH_SOCK=/s ssh-add -L"] = effecttest.Result{Out: []byte("ssh-rsa BBBB x\n")}
	if k.listed(context.Background(), "/h/k.pub", "/s") {
		t.Error("listed found a key the agent lacks")
	}
	if len(out.Lines) != 0 {
		t.Errorf("listed said %q", out.Lines)
	}
}
