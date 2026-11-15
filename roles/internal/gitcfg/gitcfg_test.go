package gitcfg_test

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/roles/internal/gitcfg"
)

func TestSettingsIsTheSharedText(t *testing.T) {
	got := gitcfg.Settings(gitcfg.Inputs{Key: "/h/k.pub",
		Gh: "/opt/gh", Program: "/h/sign"})
	want := "[user]\n\tname = \"zadykian\"\n\temail = me@zadykian.dev\n" +
		"# Each commit and tag signed with the machine's SSH key, which the allowed signers file trusts for the\n" +
		"# email above. The public key's path, so that an agent holding the key signs too; otherwise\n" +
		"# ssh-keygen reads the private key beside it.\n" +
		"\tsigningkey = \"/h/k.pub\"\n" +
		"[gpg]\n\tformat = ssh\n" +
		"[gpg \"ssh\"]\n\tallowedSignersFile = \"/etc/ws/allowed_signers\"\n\tprogram = \"/h/sign\"\n" +
		"[commit]\n\tgpgsign = true\n[tag]\n\tgpgsign = true\n" +
		"# A fetch drops the remote-tracking branches that the remote deleted, a landed pull request's\n" +
		"# say, so that Zed's and lazygit's branch lists hold none.\n" +
		"[fetch]\n\tprune = true\n" +
		"# GitHub CLI's credential helper, byte for byte as `gh auth setup-git` writes it, the space after\n" +
		"# the empty helper's = included, so that running that changes nothing here. The empty helper\n" +
		"# drops any set before it.\n" +
		"[credential \"https://github.com\"]\n\thelper = \n\thelper = !/opt/gh auth git-credential\n" +
		"[credential \"https://gist.github.com\"]\n\thelper = \n\thelper = !/opt/gh auth git-credential\n"
	if got != want {
		t.Errorf("Settings =\n%s\nwant\n%s", got, want)
	}
}

func TestSettingsQuotesTheKeyAndNamesNoProgramWithoutOne(t *testing.T) {
	got := gitcfg.Settings(gitcfg.Inputs{Key: `/h/"k".pub`})
	for _, want := range []string{`signingkey = "/h/\"k\".pub"`} {
		if !strings.Contains(got, want) {
			t.Errorf("Settings lacks %s:\n%s", want, got)
		}
	}
	if strings.Contains(got, "program") {
		t.Errorf("Settings names a program with none given:\n%s", got)
	}
}

func TestAllowedSignersTrustsTheKeyForTheEmail(t *testing.T) {
	if got, want := gitcfg.AllowedSigners("# h", "ssh-ed25519 AAAA"), "# h\nme@zadykian.dev ssh-ed25519 AAAA\n"; got != want {
		t.Errorf("AllowedSigners = %q, want %q", got, want)
	}
}
