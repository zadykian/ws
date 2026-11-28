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
	in.Program = "/etc/ws/ssh-keygen"
	if got := config(in); !strings.Contains(got, "\tprogram = \"/etc/ws/ssh-keygen\"\n") {
		t.Errorf("config lacks the program:\n%s", got)
	}
}

func TestWrapperSetsSewardsSocket(t *testing.T) {
	got := wrapper("/Users/v")
	if !strings.HasPrefix(got, "#!/bin/sh\n"+file.Header("#")+"\n") {
		t.Errorf("Wrapper = %q, want the header on line 2", got)
	}
	if !strings.Contains(got, "SSH_AUTH_SOCK='/Users/v/.ssh/seward.sock'\nexport SSH_AUTH_SOCK\nexec ssh-keygen \"$@\"\n") {
		t.Errorf("Wrapper = %q", got)
	}
}

// The texts name seward's signing key and the wrapper that holds its agent, for any user.
func TestTextsSignWithSewardsKeyThroughTheWrapper(t *testing.T) {
	texts := TextsOf("/Users/v")
	for _, want := range []string{"\tsigningkey = \"/Users/v/.ssh/ws-sign.pub\"\n", "\tprogram = \"/etc/ws/ssh-keygen\"\n"} {
		if !strings.Contains(texts.Config, want) {
			t.Errorf("config lacks %q:\n%s", want, texts.Config)
		}
	}
	if texts.Keygen != wrapper("/Users/v") {
		t.Errorf("keygen = %q, want the wrapper", texts.Keygen)
	}
}
