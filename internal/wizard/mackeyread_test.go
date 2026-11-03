package wizard

import (
	"context"
	"testing"
	"time"

	"github.com/zadykian/ws/effect/effecttest"
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

func TestMacKeySame(t *testing.T) {
	k, m, _ := newMacKey(t, "")
	m.Files["/h/k"], m.Files["/h/k.pub"] = []byte("private"), []byte(pubText)
	m.Results["ln -sf /h/k "+lockDir+"/key"] = effecttest.Result{}
	m.Results["rm -f "+lockDir+"/key"] = effecttest.Result{}
	m.Results["ssh-keygen -l -f "+lockDir+"/key"] = effecttest.Result{Out: []byte("256 SHA256:aa me (ED25519)\n")}
	m.Results["ssh-keygen -l -f /h/k.pub"] = effecttest.Result{Out: []byte("256 SHA256:aa me (ED25519)\n")}
	if !k.same(context.Background(), "/h/k") {
		t.Error("same = false for one fingerprint")
	}
	m.Results["ssh-keygen -l -f /h/k.pub"] = effecttest.Result{Out: []byte("256 SHA256:bb me (ED25519)\n")}
	if k.same(context.Background(), "/h/k") {
		t.Error("same = true for two fingerprints")
	}
	delete(m.Files, "/h/k")
	if k.same(context.Background(), "/h/k") {
		t.Error("same = true without the private key")
	}
}
