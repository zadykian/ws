package git

import "strings"

// header is the first line of each file the step writes.
const header = "# Written by ws client macos apply from ~/.config/ws/mac.conf: its next run replaces changes " +
	"made here."

// inputs are the values of ~/.config/ws/gitconfig.
type inputs struct {
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

// config returns ~/.config/ws/gitconfig: the identity, and each commit and tag signed with the key,
// through Program where there is one. The allowed signers file trusts the key for the email. Then
// gh's credential command follows, as gh auth setup-git writes it, after an empty value that drops
// any set before it. The name and the paths go in quotes, which mac.conf's values cannot end.
func config(in inputs) string {
	var b strings.Builder
	b.WriteString(header + "\n")
	b.WriteString("# ~/.gitconfig includes it at its top, so that its own settings win where both set one.\n")
	b.WriteString("[user]\n    name = \"" + in.Name + "\"\n    email = " + in.Email + "\n")
	b.WriteString("    signingkey = \"" + in.Key + "\"\n")
	b.WriteString("[commit]\n    gpgsign = true\n[tag]\n    gpgsign = true\n")
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

// allowedSigners returns the file that trusts the key for the email in git verify-commit.
func allowedSigners(email, public string) string {
	return header + "\n" + email + " " + public + "\n"
}

// wrapper returns the script that gives ssh-keygen Secretive's agent, for a key that Secretive
// holds. So a terminal and an IDE sign alike, whatever agent the app that runs git has.
func wrapper(secretive string) string {
	return "#!/bin/sh\n" +
		"# Written by ws client macos apply: git's gpg.ssh.program, ssh-keygen with Secretive's agent, which\n" +
		"# holds the signing key. Its next run replaces changes made here.\n" +
		"SSH_AUTH_SOCK='" + secretive + "/socket.ssh'\n" +
		"export SSH_AUTH_SOCK\n" +
		"exec ssh-keygen \"$@\"\n"
}
