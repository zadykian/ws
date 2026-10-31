package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
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
