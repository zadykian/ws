package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestMacKeySecretiveKey(t *testing.T) {
	const socket = "/h/Library/Containers/com.maxgoedjen.Secretive.SecretAgent/Data/socket.ssh"
	k, m, out := newMacKey(t, "WS_REMOTE=\n")
	m.Dirs["/h/Applications/Secretive.app"], m.Dirs[keys] = 0o755, 0o755
	m.Files[keys+"/ws.pub"] = []byte(p256)
	m.Results["open -a Secretive"] = effecttest.Result{}
	m.Results["mkdir -p /h/.config/ws"] = effecttest.Result{}
	m.Results["ssh-keygen -l -f "+keys+"/ws.pub"] = effecttest.Result{Out: []byte("256 SHA256:aa me\n")}
	m.Results["env SSH_AUTH_SOCK="+socket+" ssh-add -L"] = effecttest.Result{Out: []byte(p256)}
	if err := k.secretiveKey(context.Background()); err != nil {
		t.Fatalf("secretiveKey = %v", err)
	}
	want := "WS_REMOTE=\nWS_IDENTITY=~/Library/Containers/com.maxgoedjen.Secretive.SecretAgent/Data/PublicKeys/ws.pub\n" +
		"WS_IDENTITY_AGENT=~/Library/Containers/com.maxgoedjen.Secretive.SecretAgent/Data/socket.ssh\n"
	if got := string(m.Files[confPathForTest]); got != want {
		t.Errorf("mac.conf = %q, want %q", got, want)
	}
	m.Results["env SSH_AUTH_SOCK="+socket+" ssh-add -L"] = effecttest.Result{}
	if err := k.secretiveKey(context.Background()); err == nil || !strings.Contains(err.Error(), "lists no key of "+keys+"/ws.pub") {
		t.Errorf("secretiveKey where the agent lists none = %v, saying %q", err, out.Lines)
	}
}
