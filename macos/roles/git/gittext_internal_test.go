package git

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/internal/gitcfg"
)

func TestConfigSharesTheSettingsUnderItsHeader(t *testing.T) {
	in := gitcfg.Inputs{Key: "/h/k.pub", Gh: "/opt/gh"}
	got := config(in)
	for _, want := range []string{
		file.Header("#") + "\n# ~/.gitconfig includes it at its top",
		"\tname = \"zadykian\"\n\temail = me@zadykian.dev\n",
		"\tsigningkey = \"/h/k.pub\"\n",
		"[fetch]\n\tprune = true\n",
		"[credential \"https://github.com\"]\n\thelper = \n\thelper = !/opt/gh auth git-credential\n",
		"\tallowedSignersFile = \"/etc/ws/allowed_signers\"\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("config lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "program") {
		t.Errorf("config names a program with none given:\n%s", got)
	}
	in.Program = "/h/.config/ws/secretive-ssh-keygen"
	if got := config(in); !strings.Contains(got, "\tprogram = \"/h/.config/ws/secretive-ssh-keygen\"\n") {
		t.Errorf("config lacks the program:\n%s", got)
	}
}

func TestWrapperSetsSecretivesSocket(t *testing.T) {
	got := wrapper("/S/Data")
	if !strings.HasPrefix(got, "#!/bin/sh\n"+file.Header("#")+"\n") {
		t.Errorf("Wrapper = %q, want the header on line 2", got)
	}
	if !strings.Contains(got, "SSH_AUTH_SOCK='/S/Data/socket.ssh'\nexport SSH_AUTH_SOCK\nexec ssh-keygen \"$@\"\n") {
		t.Errorf("Wrapper = %q", got)
	}
}
