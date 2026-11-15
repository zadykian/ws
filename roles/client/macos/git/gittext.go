package git

import "github.com/zadykian/ws/roles/internal/gitcfg"

// header is the first line of each file the step writes.
const header = "# Written by ws client macos apply from /etc/ws/values.yml: its next run replaces changes " +
	"made here."

// config returns /etc/ws/gitconfig: its header, and the settings that ws shares with the server.
// ~/.gitconfig includes it at its top, so that its own settings win.
func config(in gitcfg.Inputs) string {
	return header + "\n" +
		"# ~/.gitconfig includes it at its top, so that its own settings win where both set one.\n" +
		gitcfg.Settings(in)
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
