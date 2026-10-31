package gitcfg_test

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/gitcfg"
)

func TestConfigSignsWithTheKey(t *testing.T) {
	in := gitcfg.Inputs{Name: "A B", Email: "a@b.c", Key: "/h/k.pub", Gh: "/opt/gh", Dir: "/h/.config/ws"}
	got := gitcfg.Config(in)
	for _, want := range []string{
		"    name = \"A B\"\n    email = a@b.c\n    signingkey = \"/h/k.pub\"\n",
		"[commit]\n    gpgsign = true\n",
		"[credential \"https://github.com\"]\n    helper =\n    helper = !/opt/gh auth git-credential\n",
		"[credential \"https://gist.github.com\"]\n    helper =\n",
		"    allowedSignersFile = \"/h/.config/ws/allowed_signers\"\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Config lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "program") {
		t.Errorf("Config names a program with none given:\n%s", got)
	}
	in.Program = "/h/.config/ws/secretive-ssh-keygen"
	if got := gitcfg.Config(in); !strings.HasSuffix(got, "    program = \"/h/.config/ws/secretive-ssh-keygen\"\n") {
		t.Errorf("Config ends %q", got)
	}
}

func TestAllowedSignersTrustsTheKeyForTheEmail(t *testing.T) {
	got := gitcfg.AllowedSigners("a@b.c", "ssh-ed25519 AAAA")
	if !strings.HasSuffix(got, "\na@b.c ssh-ed25519 AAAA\n") || !strings.HasPrefix(got, gitcfg.Header) {
		t.Errorf("AllowedSigners = %q", got)
	}
}

func TestWrapperSetsSecretivesSocket(t *testing.T) {
	got := gitcfg.Wrapper("/S/Data")
	if !strings.Contains(got, "SSH_AUTH_SOCK='/S/Data/socket.ssh'\nexport SSH_AUTH_SOCK\nexec ssh-keygen \"$@\"\n") {
		t.Errorf("Wrapper = %q", got)
	}
}
