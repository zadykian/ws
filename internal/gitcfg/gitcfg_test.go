package gitcfg_test

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/internal/gitcfg"
)

func TestSettingsIsTheSharedText(t *testing.T) {
	got := gitcfg.Settings(gitcfg.Mac(), gitcfg.Inputs{Key: "/h/k.pub",
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

func TestEachSideHasItsOwnIdentity(t *testing.T) {
	if got, want := gitcfg.Mac(), (gitcfg.Identity{Name: "zadykian", Email: "me@zadykian.dev"}); got != want {
		t.Errorf("Mac = %v, want %v", got, want)
	}
	if got, want := gitcfg.WS(), (gitcfg.Identity{Name: "zadykiand", Email: "su@zadykian.dev"}); got != want {
		t.Errorf("WS = %v, want %v", got, want)
	}
	got := gitcfg.Settings(gitcfg.WS(), gitcfg.Inputs{Key: "/h/k.pub"})
	if want := "[user]\n\tname = \"zadykiand\"\n\temail = su@zadykian.dev\n"; !strings.HasPrefix(got, want) {
		t.Errorf("Settings of ws = %q, want it to open with %q", got, want)
	}
}

func TestSettingsQuotesTheKeyAndNamesNoProgramWithoutOne(t *testing.T) {
	got := gitcfg.Settings(gitcfg.Mac(), gitcfg.Inputs{Key: `/h/"k".pub`})
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
	for id, want := range map[gitcfg.Identity]string{
		gitcfg.Mac(): "# h\nme@zadykian.dev ssh-ed25519 AAAA\n",
		gitcfg.WS():  "# h\nsu@zadykian.dev ssh-ed25519 AAAA\n",
	} {
		if got := gitcfg.AllowedSigners(id, "# h", "ssh-ed25519 AAAA"); got != want {
			t.Errorf("AllowedSigners(%v) = %q, want %q", id, got, want)
		}
	}
}
