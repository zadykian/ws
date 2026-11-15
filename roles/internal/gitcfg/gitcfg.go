// Package gitcfg holds git's settings that ws and the Mac both write, as pure text.
package gitcfg

import (
	"strings"

	"github.com/zadykian/ws/roles/internal/etcws"
)

// Name and Email are the identity of every commit, on ws and on the Mac alike.
const (
	Name  = "zadykian"
	Email = "me@zadykian.dev"
)

// Signers is the allowed signers file, root's, the same path on both sides.
const Signers = etcws.Dir + "/allowed_signers"

// Inputs are the values of the settings.
type Inputs struct {
	// Key is the signing key's .pub and Gh GitHub's CLI, each as an absolute path. Program is
	// gpg.ssh.program, or empty to sign with ssh-keygen as found.
	Key, Gh, Program string
}

// Settings returns the sections of identity, signing, fetch and credentials, with tabs. The name
// and the paths go in quotes. The email and gh's command stay bare, as gh auth setup-git writes them.
func Settings(in Inputs) string {
	var b strings.Builder
	b.WriteString("[user]\n\tname = " + quote(Name) + "\n\temail = " + Email + "\n")
	b.WriteString("# Each commit and tag signed with the machine's SSH key, which the allowed signers file " +
		"trusts for the\n# email above. The public key's path, so that an agent holding the key signs " +
		"too; otherwise\n# ssh-keygen reads the private key beside it.\n")
	b.WriteString("\tsigningkey = " + quote(in.Key) + "\n")
	b.WriteString("[gpg]\n\tformat = ssh\n")
	b.WriteString("[gpg \"ssh\"]\n\tallowedSignersFile = " + quote(Signers) + "\n")
	if in.Program != "" {
		b.WriteString("\tprogram = " + quote(in.Program) + "\n")
	}
	b.WriteString("[commit]\n\tgpgsign = true\n[tag]\n\tgpgsign = true\n")
	b.WriteString("# A fetch drops the remote-tracking branches that the remote deleted, a landed pull " +
		"request's\n# say, so that Zed's and lazygit's branch lists hold none.\n")
	b.WriteString("[fetch]\n\tprune = true\n")
	b.WriteString("# GitHub CLI's credential helper, byte for byte as `gh auth setup-git` writes it, the " +
		"space after\n# the empty helper's = included, so that running that changes nothing here. The " +
		"empty helper\n# drops any set before it.\n")
	for _, host := range []string{"github.com", "gist.github.com"} {
		b.WriteString("[credential \"https://" + host + "\"]\n\thelper = \n")
		b.WriteString("\thelper = !" + in.Gh + " auth git-credential\n")
	}
	return b.String()
}

// AllowedSigners returns the file that trusts the key for Email, under its header.
func AllowedSigners(header, key string) string {
	return header + "\n" + Email + " " + key + "\n"
}

// quote puts s in quotes for a git config file, with \ and " escaped.
func quote(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}
