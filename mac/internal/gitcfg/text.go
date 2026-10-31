// Package gitcfg holds the texts of the git step of ws setup, as pure functions: the files it
// writes, and what it makes of git's answers.
package gitcfg

import "strings"

// Header is the first line of each file the step writes.
const Header = "# Written by ws's mac/setup.sh from ~/.config/ws/mac.conf: its next run replaces changes " +
	"made here."

// Inputs are the values of ~/.config/ws/gitconfig.
type Inputs struct {
	Name, Email string
	// Key is the signing key's .pub, as an absolute path.
	Key string
	// Gh is the absolute path of GitHub's CLI.
	Gh string
	// Dir is ~/.config/ws, where the step writes its files.
	Dir string
	// Program is gpg.ssh.program, or empty to sign with ssh-keygen as found.
	Program string
}

// Config returns ~/.config/ws/gitconfig: the identity, and each commit signed with the key, which
// the allowed signers file trusts for the email, through Program where there is one. Then gh's
// credential command, as gh auth setup-git writes it, after an empty value that drops any set before
// it. The name and the paths go in quotes, which mac.conf's values cannot end.
func Config(in Inputs) string {
	var b strings.Builder
	b.WriteString(Header + "\n")
	b.WriteString("# ~/.gitconfig includes it at its top, so that its own settings win where both set one.\n")
	b.WriteString("[user]\n    name = \"" + in.Name + "\"\n    email = " + in.Email + "\n")
	b.WriteString("    signingkey = \"" + in.Key + "\"\n")
	b.WriteString("[commit]\n    gpgsign = true\n")
	for _, host := range []string{"github.com", "gist.github.com"} {
		b.WriteString("[credential \"https://" + host + "\"]\n    helper =\n")
		b.WriteString("    helper = !" + in.Gh + " auth git-credential\n")
	}
	b.WriteString("[gpg]\n    format = ssh\n")
	b.WriteString("[gpg \"ssh\"]\n    allowedSignersFile = \"" + in.Dir + "/allowed_signers\"\n")
	if in.Program != "" {
		b.WriteString("    program = \"" + in.Program + "\"\n")
	}
	return b.String()
}

// AllowedSigners returns the file that trusts the key for the email in git verify-commit.
func AllowedSigners(email, public string) string {
	return Header + "\n" + email + " " + public + "\n"
}

// Wrapper returns the script that gives ssh-keygen Secretive's agent, for a key that Secretive
// holds. So a terminal and an IDE sign alike, whatever agent the app that runs git has.
func Wrapper(secretive string) string {
	return "#!/bin/sh\n" +
		"# Written by ws's mac/setup.sh: git's gpg.ssh.program, ssh-keygen with Secretive's agent, which\n" +
		"# holds the signing key. Its next run replaces changes made here.\n" +
		"SSH_AUTH_SOCK='" + secretive + "/socket.ssh'\n" +
		"export SSH_AUTH_SOCK\n" +
		"exec ssh-keygen \"$@\"\n"
}
