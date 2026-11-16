package git

import (
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/roles/internal/gitcfg"
)

// config returns /etc/ws/gitconfig: its header, and the settings that ws shares with the server.
// ~/.gitconfig includes it at its top, so that its own settings win.
func config(in gitcfg.Inputs) string {
	return file.Header("#") + "\n" +
		"# ~/.gitconfig includes it at its top, so that its own settings win where both set one.\n" +
		gitcfg.Settings(in)
}

// wrapper returns the script that gives ssh-keygen Secretive's agent, for a key that Secretive
// holds. So a terminal and an IDE sign alike, whatever agent the app that runs git has.
func wrapper(secretive string) string {
	return "#!/bin/sh\n" +
		file.Header("#") + "\n" +
		"# git's gpg.ssh.program: ssh-keygen with Secretive's agent, which holds the signing key.\n" +
		"SSH_AUTH_SOCK='" + secretive + "/socket.ssh'\n" +
		"export SSH_AUTH_SOCK\n" +
		"exec ssh-keygen \"$@\"\n"
}
