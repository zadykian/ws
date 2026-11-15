package git

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/roles/internal/gitcfg"
)

func TestConfigSharesTheSettingsUnderItsHeader(t *testing.T) {
	in := gitcfg.Inputs{Name: "A B", Email: "a@b.c", Key: "/h/k.pub", Signers: "/h/.config/ws/allowed_signers",
		Gh: "/opt/gh"}
	got := config(in)
	for _, want := range []string{
		header + "\n# ~/.gitconfig includes it at its top",
		"\tname = \"A B\"\n\temail = a@b.c\n",
		"\tsigningkey = \"/h/k.pub\"\n",
		"[fetch]\n\tprune = true\n",
		"[credential \"https://github.com\"]\n\thelper = \n\thelper = !/opt/gh auth git-credential\n",
		"\tallowedSignersFile = \"/h/.config/ws/allowed_signers\"\n",
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
	if !strings.Contains(got, "SSH_AUTH_SOCK='/S/Data/socket.ssh'\nexport SSH_AUTH_SOCK\nexec ssh-keygen \"$@\"\n") {
		t.Errorf("Wrapper = %q", got)
	}
}
