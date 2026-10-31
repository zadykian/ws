package wizard

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
)

func planned(k *MacKey) string {
	p, err := k.Plan(context.Background())
	return fmt.Sprint(p.Changes, err)
}

func TestMacKeyPlan(t *testing.T) {
	const conf = "WS_IDENTITY=~/k.pub\nWS_IDENTITY_AGENT=/s\n"
	k, m, _ := newMacKey(t, conf)
	m.Files["/h/k.pub"] = []byte(pubText)
	m.Results["ssh-keygen -l -f /h/k.pub"] = effecttest.Result{Out: []byte("256 SHA256:aa me\n")}
	m.Results["env SSH_AUTH_SOCK=/s ssh-add -L"] = effecttest.Result{Out: []byte("ssh-ed25519 AAAAC3 x\n")}
	if got := planned(k); got != "[] <nil>" {
		t.Errorf("a listed key: Plan = %q", got)
	}
	m.Results["env SSH_AUTH_SOCK=/s ssh-add -L"] = effecttest.Result{Out: []byte("ssh-rsa BBBB x\n")}
	if got := planned(k); !strings.Contains(got, "the agent at /s lists no key of /h/k.pub") {
		t.Errorf("a key the agent lacks: Plan = %q", got)
	}
	k, _, _ = newMacKey(t, "")
	if got := planned(k); !strings.Contains(got, "would ask whether Secretive keeps the Mac's key") {
		t.Errorf("no key: Plan = %q", got)
	}
}
