package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestMacKeySecretiveKey(t *testing.T) {
	k, m, _ := newMacKey(t, "WS_REMOTE=\n")
	m.Dirs["/h/Applications/Secretive.app"] = 0o755
	agent(m, p256)
	m.Results["open -a Secretive"] = effecttest.Result{}
	m.Results["mkdir -p /h/.config/ws"] = effecttest.Result{}
	if err := k.secretiveKey(context.Background()); err != nil {
		t.Fatalf("secretiveKey = %v", err)
	}
	want := "WS_REMOTE=\nWS_IDENTITY=" + k.tilde(copyOf(p256)) + "\n" +
		"WS_IDENTITY_AGENT=~/Library/Containers/com.maxgoedjen.Secretive.SecretAgent/Data/socket.ssh\n"
	if got := string(m.Files[confPathForTest]); got != want {
		t.Errorf("mac.conf = %q, want %q", got, want)
	}
	if string(m.Files[copyOf(p256)]) != p256 {
		t.Errorf("the copy holds %q", m.Files[copyOf(p256)])
	}
}

func TestMacKeySecretiveKeyStopsWhereItsAgentLacksTheKey(t *testing.T) {
	k, m, out := newMacKey(t, "WS_IDENTITY=~/k.pub\nWS_IDENTITY_AGENT=/s\n")
	m.Dirs["/h/Applications/Secretive.app"] = 0o755
	agent(m, p256)
	m.Files["/h/k.pub"] = []byte(pubText)
	m.Results["ssh-keygen -l -f /h/k.pub"] = effecttest.Result{Out: []byte("256 SHA256:aa me\n")}
	m.Results["open -a Secretive"] = effecttest.Result{}
	m.Results["env SSH_AUTH_SOCK=/s ssh-add -L"] = effecttest.Result{}
	if err := k.secretiveKey(context.Background()); err == nil || !strings.Contains(err.Error(), "lists no key of /h/k.pub") {
		t.Errorf("secretiveKey where the agent lists none = %v, saying %q", err, out.Lines)
	}
}
