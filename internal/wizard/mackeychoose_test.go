package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestMacKeyChooseWithNoTerminal(t *testing.T) {
	k, _, _ := newMacKey(t, "")
	err := k.choose(context.Background(), chooseKey{pub: "/h/.ssh/id_ed25519.pub"})
	if err == nil || !strings.HasPrefix(err.Error(), "Mac key: no terminal to ask whether Secretive keeps") {
		t.Errorf("choose = %v", err)
	}
}

func TestMacKeyChooseNo(t *testing.T) {
	k, m, _ := newMacKey(t, "WS_REMOTE=\n")
	k.Tty = true
	k.P.(*effecttest.Prompt).Answers = map[string]string{"Keep the Mac's ssh key in Secretive, in the Secure Enclave? [Y/n]": "no"}
	m.Results["ssh-keygen -q -t ed25519 -a 100 -f /h/.ssh/id_ed25519"] = effecttest.Result{}
	m.Results["ssh-add --apple-use-keychain /h/.ssh/id_ed25519"] = effecttest.Result{}
	m.Results["mkdir -p /h/.config/ws"] = effecttest.Result{}
	if err := k.choose(context.Background(), chooseKey{pub: "/h/.ssh/id_ed25519.pub"}); err != nil {
		t.Fatalf("choose = %v", err)
	}
	if got := string(m.Files[confPathForTest]); got != "WS_REMOTE=\nWS_IDENTITY=~/.ssh/id_ed25519.pub\n" {
		t.Errorf("mac.conf = %q", got)
	}
}

func TestMacKeyGit(t *testing.T) {
	k, m, out := newMacKey(t, "WS_GIT_NAME=ada\nWS_GIT_EMAIL=a@b.c\nWS_IDENTITY=~/k.pub\nWS_IDENTITY_AGENT=/s\n")
	m.Dirs[keys] = 0o755
	m.Files["/h/k.pub"], m.Files[keys+"/git.pub"] = []byte(pubText), []byte(p256)
	m.Results["ssh-keygen -l -f /h/k.pub"] = effecttest.Result{Out: []byte("256 SHA256:aa me\n")}
	m.Results["open -a Secretive"] = effecttest.Result{}
	m.Results["mkdir -p /h/.config/ws"] = effecttest.Result{}
	if err := k.git(context.Background(), true); err != nil {
		t.Fatalf("git = %v, saying %q", err, out.Lines)
	}
	if got := string(m.Files[confPathForTest]); !strings.HasSuffix(got, "\nWS_GIT_SIGNING_KEY=~/Library/Containers/com.maxgoedjen.Secretive.SecretAgent/Data/PublicKeys/git.pub\n") {
		t.Errorf("mac.conf = %q", got)
	}
	if err := k.git(context.Background(), true); err != nil {
		t.Errorf("git again = %v", err)
	}
}
